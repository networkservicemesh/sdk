// Copyright (c) 2020 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package connect is intended to allow passthrough style Endpoints to have a server that also connects to a client
package connect

import (
	"context"
	"net/url"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type closeFunc func() bool

type connectServer struct {
	ctx               context.Context
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clientDialOptions []grpc.DialOption
	connInfos         connectionInfoMap
	clients           clientmap.RefcountMap
	clientsMutex      sync.Mutex
	clientsCloseFuncs map[networkservice.NetworkServiceClient]closeFunc
}

type connectionInfo struct {
	clientURL *url.URL
	client    networkservice.NetworkServiceClient
}

// NewServer - chain element that
func NewServer(
	ctx context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient,
	clientDialOptions ...grpc.DialOption,
) networkservice.NetworkServiceServer {
	return &connectServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialOptions: clientDialOptions,
		clientsCloseFuncs: make(map[networkservice.NetworkServiceClient]closeFunc),
	}
}

func (s *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clientURL := clienturlctx.ClientURL(ctx)

	client, onClose := s.client(ctx, request.GetConnection())
	conn, err := client.Request(ctx, request.Clone())
	if err != nil {
		if _, ok := s.connInfos.Load(request.GetConnection().GetId()); !ok && onClose != nil {
			s.closeClient(client)
		}

		// close current client chain if grpc connection was closed
		if grpcutils.UnwrapCode(err) == codes.Canceled {
			s.deleteClient(client)
			s.connInfos.Delete(request.GetConnection().GetId())
		}

		return nil, err
	}

	// Update request.Connection
	request.Connection = conn

	s.connInfos.Store(conn.GetId(), connectionInfo{
		clientURL: clientURL,
		client:    client,
	})

	// Carry on with next.Server
	return next.Server(ctx).Request(ctx, request)
}

func (s *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if connInfo, ok := s.connInfos.LoadAndDelete(conn.GetId()); ok {
		_, clientErr = connInfo.client.Close(ctx, conn)
		s.closeClient(connInfo.client)
	}

	_, err := next.Server(ctx).Close(ctx, conn)

	if clientErr != nil && err != nil {
		return nil, errors.Wrapf(err, "errors during client close: %v", clientErr)
	}
	if clientErr != nil {
		return nil, errors.Wrap(clientErr, "errors during client close")
	}
	return &empty.Empty{}, err
}

func (s *connectServer) client(ctx context.Context, conn *networkservice.Connection) (networkservice.NetworkServiceClient, closeFunc) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	logEntry := logger.Log(ctx).WithField("connectServer", "client")

	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		err := errors.Errorf("clientURL not found for incoming connection: %+v", conn)
		return injecterror.NewClient(err), nil
	}

	// First check if we have already requested some clientURL with this conn.GetID().
	if connInfo, ok := s.connInfos.Load(conn.GetId()); ok {
		if *connInfo.clientURL == *clientURL {
			return connInfo.client, nil
		}
		// For some reason we have changed the clientURL, so we need to close and delete the existing client.
		if _, clientErr := connInfo.client.Close(ctx, conn); clientErr != nil {
			logEntry.Warnf("failed to close client: %+v", clientErr)
		}

		if onClose, ok := s.clientsCloseFuncs[connInfo.client]; ok {
			onClose()
		} else {
			logEntry.Warnf("failed to delete client connected to %s", connInfo.clientURL.String())
		}
	}

	// Fast path if we already have client for the clientURL and we chould not reconnect, use it.
	client, loaded := s.clients.Load(clientURL.String())
	var onClose closeFunc
	if !loaded {
		// If not, create and LoadOrStore a new one.
		client, onClose = s.newClient(clientURL)
		s.clients.Store(clientURL.String(), client)
		s.clientsCloseFuncs[client] = onClose
	}

	return client, onClose
}

// newClient has to be called under clientsMutex protection
func (s *connectServer) newClient(clientURL *url.URL) (networkservice.NetworkServiceClient, closeFunc) {
	clientPtr := new(networkservice.NetworkServiceClient)

	ctx, cancel := context.WithCancel(s.ctx)
	onClose := func() bool {
		client, ok := s.clients.Load(clientURL.String())
		if ok {
			s.clients.Delete(clientURL.String())
		}

		if client != *clientPtr {
			cancel()
			delete(s.clientsCloseFuncs, *clientPtr)
			return true
		}

		if deleted := s.clients.Delete(clientURL.String()); deleted {
			cancel()
			delete(s.clientsCloseFuncs, *clientPtr)
			return true
		}

		return false
	}

	*clientPtr = clienturl.NewClient(clienturlctx.WithClientURL(ctx, clientURL), s.clientFactory, s.clientDialOptions...)

	return *clientPtr, onClose
}

func (s *connectServer) closeClient(client networkservice.NetworkServiceClient) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if onClose, ok := s.clientsCloseFuncs[client]; ok {
		onClose()
	}
}

func (s *connectServer) deleteClient(client networkservice.NetworkServiceClient) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	if onClose, ok := s.clientsCloseFuncs[client]; ok {
		for {
			if onClose() {
				return
			}
		}
	}
}
