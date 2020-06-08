// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package callback - provide GRPC API to perform client callbacks.
package callback

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ClientListener - inform server about new client is arrived or disconnected.
type ClientListener interface {
	ClientConnected(id string)
	ClientDisconnected(id string)
}

// Server - a callback server, hold client callback connections and allow to provide client
type Server interface {
	AddListener(listener ClientListener)
	CallbackServiceServer
}

type serverImpl struct {
	connections      map[string]*serverClientConnImpl
	listeners        []ClientListener
	lock             sync.Mutex
	identityProvider IdentityProvider
}

// IdentityProvider - A function to retrieve identity from grpc connection context and create clients based on it.
type IdentityProvider func(ctx context.Context) (string, error)

// IdentityByAuthority - return identity by :authority
func IdentityByAuthority(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		err := errors.New("Not metadata provided")
		logrus.Error(err)
		return "", err
	}
	return md.Get(":authority")[0], nil
}

// NewServer - creates a new callback server to handle clients, should be used to create client connections back to client.
func NewServer(provider IdentityProvider) Server {
	return &serverImpl{
		identityProvider: provider,
		connections:      map[string]*serverClientConnImpl{},
	}
}

// WithCallbackDialer - return a grpc.DialOption with callback server inside to perform a dial
func WithCallbackDialer(server CallbackServiceServer, target string) grpc.DialOption {
	dialFunc, err := server.(*serverImpl).dial(target)
	if err != nil {
		logrus.Errorf("Failed to connect to callback: %v", err)
	}
	return grpc.WithContextDialer(dialFunc)
}

// AddListener - add listener to client, to be informed abount new clients are joined.
func (s *serverImpl) AddListener(listener ClientListener) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.listeners = append(s.listeners, listener)
	for k := range s.connections {
		listener.ClientConnected(k)
	}
}

// HandleCallbacks - main entry point for server, handle client stream here.
func (s *serverImpl) HandleCallbacks(serverClient CallbackService_HandleCallbacksServer) error {
	clientID, err := s.identityProvider(serverClient.Context())
	if err != nil {
		// Return error in case we could not understand peer id
		return err
	}

	ctx, cancelFunc := context.WithCancel(serverClient.Context())
	defer cancelFunc()

	handleID := fmt.Sprintf("callback:%v", clientID)

	conn, addErr := s.addConnection(handleID, serverClient)
	if addErr != nil {
		// This could be duplicate connection id
		return addErr
	}
	conn.lock.Lock()
	conn.id = handleID
	conn.serverCtx = ctx
	conn.cancel = cancelFunc
	conn.lock.Unlock()

	defer s.removeConnection(handleID)

	// Wait until context will be complete.
	<-ctx.Done()
	return ctx.Err()
}

func (s *serverImpl) addConnection(key string, serverClient CallbackService_HandleCallbacksServer) (*serverClientConnImpl, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.connections[key]; ok {
		// Connection already defined with this ID.
		return nil, errors.Errorf("connection is already active, ID %v is duplicate.", key)
	}
	conn := &serverClientConnImpl{
		server: serverClient,
	}
	s.connections[key] = conn
	// Notify listeners
	for _, l := range s.listeners {
		l.ClientConnected(key)
	}
	return conn, nil
}

func (s *serverImpl) removeConnection(key string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.connections, key)
	for _, l := range s.listeners {
		l.ClientDisconnected(key)
	}
}

type serverClientConnImpl struct {
	server    CallbackService_HandleCallbacksServer
	created   bool
	cancel    context.CancelFunc
	serverCtx context.Context
	id        string
	lock      sync.Mutex
}

func (s *serverImpl) dial(target string) (func(context.Context, string) (net.Conn, error), error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	srv, ok := s.connections[target]
	if ok {
		if srv.created {
			return nil, errors.New("Client is already created")
		}
		srv.created = true
		return func(ctx context.Context, target string) (net.Conn, error) {
			srv.lock.Lock()
			defer srv.lock.Unlock()
			return newConnection(ctx, srv.cancel, srv.server), nil
		}, nil
	}
	// Fall over to default dial code.
	return nil, nil
}
