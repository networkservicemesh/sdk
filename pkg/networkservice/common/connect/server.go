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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

type connectServer struct {
	ctx               context.Context
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clientDialOptions []grpc.DialOption

	connInfos connectionInfoMap
	clients   clientInfoMap
	executor  multiexecutor.MultiExecutor
}

type clientInfo struct {
	client  networkservice.NetworkServiceClient
	count   int
	onClose context.CancelFunc
}

type connectionInfo struct {
	clientURL *url.URL
	client    *clientInfo
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
	}
}

func (s *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming connection: %+v", request.GetConnection())
	}

	clientInfo := s.client(ctx, request.GetConnection())
	conn, err := clientInfo.client.Request(ctx, request.Clone())
	if err != nil {
		if _, ok := s.connInfos.Load(request.GetConnection().GetId()); !ok {
			s.closeClient(clientInfo, clientURL.String())
		}

		// close current client chain if grpc connection was closed
		if grpcutils.UnwrapCode(err) == codes.Canceled {
			s.deleteClient(clientInfo, clientURL.String())
			s.connInfos.Delete(request.GetConnection().GetId())
		}

		return nil, err
	}

	// Update request.Connection
	request.Connection = conn

	s.connInfos.Store(conn.GetId(), connectionInfo{
		clientURL: clientURL,
		client:    clientInfo,
	})

	// Carry on with next.Server
	return next.Server(ctx).Request(ctx, request)
}

func (s *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if connInfo, ok := s.connInfos.LoadAndDelete(conn.GetId()); ok {
		_, clientErr = connInfo.client.client.Close(ctx, conn)
		s.closeClient(connInfo.client, connInfo.clientURL.String())
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

func (s *connectServer) client(ctx context.Context, conn *networkservice.Connection) *clientInfo {
	logger := log.FromContext(ctx).WithField("connectServer", "client")
	clientURL := clienturlctx.ClientURL(ctx)

	// First check if we have already requested some clientURL with this conn.GetID().
	if connInfo, ok := s.connInfos.Load(conn.GetId()); ok {
		if *connInfo.clientURL == *clientURL {
			return connInfo.client
		}

		// For some reason we have changed the clientURL, so we need to close and delete the existing client.
		if _, clientErr := connInfo.client.client.Close(ctx, conn); clientErr != nil {
			logger.Warnf("failed to close client: %+v", clientErr)
		}

		s.closeClient(connInfo.client, connInfo.clientURL.String())
	}

	var client *clientInfo
	<-s.executor.AsyncExec(clientURL.String(), func() {
		// Fast path if we already have client for the clientURL and we chould not reconnect, use it.
		var loaded bool
		client, loaded = s.clients.Load(clientURL.String())
		if !loaded {
			// If not, create and LoadOrStore a new one.
			client = s.newClient(clientURL)
			s.clients.Store(clientURL.String(), client)
		}
		client.count++
	})

	return client
}

// newClient has to be called under clientsMutex protection
func (s *connectServer) newClient(clientURL *url.URL) *clientInfo {
	ctx, cancel := context.WithCancel(s.ctx)
	client := clienturl.NewClient(clienturlctx.WithClientURL(ctx, clientURL), s.clientFactory, s.clientDialOptions...)
	return &clientInfo{
		client:  client,
		count:   0,
		onClose: cancel,
	}
}

func (s *connectServer) closeClient(client *clientInfo, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		client.count--
		if client.count == 0 {
			if loadedClient, ok := s.clients.Load(clientURL); ok && client == loadedClient {
				s.clients.Delete(clientURL)
			}
			client.onClose()
		}
	})
}

func (s *connectServer) deleteClient(client *clientInfo, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		if loadedClient, ok := s.clients.Load(clientURL); ok && client == loadedClient {
			s.clients.Delete(clientURL)
		}
		client.onClose()
	})
}
