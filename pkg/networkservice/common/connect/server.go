// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/tools/clientmap"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type connectServer struct {
	ctx                      context.Context
	translationClientFactory func() networkservice.NetworkServiceClient
	clientFactory            func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient
	clientDialOptions        []grpc.DialOption
	cluentURLs               clientURLMap
	clients                  clientmap.RefcountMap
}

// NewServer - chain element that
func NewServer(
	ctx context.Context,
	translationClientFactory func() networkservice.NetworkServiceClient,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient,
	clientDialOptions ...grpc.DialOption,
) networkservice.NetworkServiceServer {
	return &connectServer{
		ctx:                      ctx,
		translationClientFactory: translationClientFactory,
		clientFactory:            clientFactory,
		clientDialOptions:        clientDialOptions,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := c.client(ctx, request.GetConnection()).Request(ctx, request.Clone())
	if err != nil {
		return nil, err
	}

	// Update request.Connection
	request.Connection = conn

	// Carry on with next.Server
	return next.Server(ctx).Request(ctx, request)
}

func (c *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	var clientErr error
	if clientURL, ok := c.cluentURLs.Load(conn.GetId()); ok {
		if client, ok := c.clients.LoadAndDelete(clientURL.String()); ok {
			_, clientErr = client.Close(ctx, conn)
		}
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

func (c *connectServer) client(ctx context.Context, conn *networkservice.Connection) networkservice.NetworkServiceClient {
	logEntry := log.Entry(ctx).WithField("connectServer", "client")

	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		err := errors.Errorf("clientURL not found for incoming connection: %+v", conn)
		return injecterror.NewClient(err)
	}

	// First check if we have already requested some clientURL with this conn.GetID().
	if oldClientURL, ok := c.cluentURLs.Load(conn.GetId()); oldClientURL != clientURL {
		if ok {
			// For some reason we have changed the clientURL, so we need to close the existing client.
			if client, ok := c.clients.LoadAndDelete(oldClientURL.String()); ok {
				if _, clientErr := client.Close(ctx, conn); clientErr != nil {
					logEntry.Warnf("failed to close client: %+v", clientErr)
				}
			}
		}
		c.cluentURLs.Store(conn.GetId(), clientURL)
	}

	// Fast path if we already have client for the clientURL, use it.
	client, ok := c.clients.Load(clientURL.String())
	if !ok {
		// If not, create and LoadOrStore a new one.
		client, _ = c.clients.LoadOrStore(clientURL.String(), chain.NewNetworkServiceClient(
			c.translationClientFactory(),
			// Note: clienturl.NewClient(...) will get properly cleaned up when dereferences.
			clienturl.NewClient(clienturlctx.WithClientURL(c.ctx, clientURL), c.clientFactory, c.clientDialOptions...),
		))
	}

	return client
}
