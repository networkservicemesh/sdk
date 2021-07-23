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
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cancelctx"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

// ClientFactory is used to created new clients when new connection is created.
type ClientFactory = func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient

type connectServer struct {
	ctx               context.Context
	clientFactory     ClientFactory
	clientDialTimeout time.Duration
	clientDialOptions []grpc.DialOption

	connInfos connectionInfoMap
	clients   clientInfoMap
	executor  multiexecutor.MultiExecutor
}

type clientInfo struct {
	client  *connectClient
	count   int
	onClose context.CancelFunc
}

type connectionInfo struct {
	clientURL *url.URL
	client    *clientInfo
}

// NewServer - server chain element that creates client subchains and requests them selecting by
//             clienturlctx.ClientURL(ctx)
func NewServer(
	ctx context.Context,
	clientFactory ClientFactory,
	options ...Option,
) networkservice.NetworkServiceServer {
	s := &connectServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialTimeout: time.Second,
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

func (s *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming connection: %+v", request.GetConnection())
	}

	c := s.client(ctx, request.GetConnection())
	if err := c.client.ctx.Err(); err != nil {
		s.deleteClient(c, clientURL.String())
		s.connInfos.Delete(request.GetConnection().GetId())
		return nil, err
	}

	conn, err := c.client.Request(ctx, request.Clone())
	if err != nil {
		if _, ok := s.connInfos.Load(request.GetConnection().GetId()); !ok {
			s.closeClient(c, clientURL.String())
		}
		return nil, err
	}

	// Update request.Connection
	request.Connection = conn

	_, refreshRequest := s.connInfos.Load(conn.GetId())
	s.connInfos.Store(conn.GetId(), connectionInfo{
		clientURL: clientURL,
		client:    c,
	})

	// Carry on with next.Server
	conn, err = next.Server(ctx).Request(ctx, request)
	// Close connection if next.Server Request finished with error
	if err != nil && !refreshRequest {
		_, cErr := s.Close(ctx, request.Connection.Clone())
		if cErr != nil {
			err = errors.Wrapf(cErr, "connection closed with error: %v", cErr)
		}
	}
	return conn, err
}

func (s *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if connInfo, ok := s.connInfos.LoadAndDelete(conn.GetId()); ok {
		_, clientErr = connInfo.client.client.Close(ctx, conn)
		s.closeClient(connInfo.client, connInfo.clientURL.String())
	} else {
		clientErr = errors.Errorf("no client found for the connection: %s", conn.GetId())
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

	var c *clientInfo
	<-s.executor.AsyncExec(clientURL.String(), func() {
		// Fast path if we already have client for the clientURL and we chould not reconnect, use it.
		var loaded bool
		c, loaded = s.clients.Load(clientURL.String())
		if !loaded {
			// If not, create and LoadOrStore a new one.
			c = s.newClient(clientURL)
			s.clients.Store(clientURL.String(), c)
		}
		c.count++
	})
	return c
}

func (s *connectServer) newClient(clientURL *url.URL) *clientInfo {
	ctx, cancel := cancelctx.WithCancel(s.ctx)
	return &clientInfo{
		client: &connectClient{
			ctx:           clienturlctx.WithClientURL(ctx, clientURL),
			dialTimeout:   s.clientDialTimeout,
			clientFactory: s.clientFactory,
			dialOptions:   s.clientDialOptions,
		},
		count:   0,
		onClose: cancel,
	}
}

func (s *connectServer) closeClient(c *clientInfo, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		c.count--
		if c.count == 0 {
			if loadedClient, ok := s.clients.Load(clientURL); ok && c == loadedClient {
				s.clients.Delete(clientURL)
			}
			c.onClose()
		}
	})
}

func (s *connectServer) deleteClient(c *clientInfo, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		if loadedClient, ok := s.clients.Load(clientURL); ok && c == loadedClient {
			s.clients.Delete(clientURL)
		}
		c.onClose()
	})
}
