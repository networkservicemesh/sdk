// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

// Package connect provides chain elements to 'connect' clients
package connect

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type connectNSServer struct {
	client      registry.NetworkServiceRegistryClient
	callOptions []grpc.CallOption
}

func (c *connectNSServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	clientResp, clientErr := c.client.Register(ctx, in, c.callOptions...)
	if clientErr != nil {
		return nil, clientErr
	}

	serverResp, serverErr := next.NetworkServiceRegistryServer(ctx).Register(ctx, clientResp)
	if serverErr != nil {
		closeCtx, closeCancel := closeCtxFunc()
		defer closeCancel()
		_, _ = c.client.Unregister(closeCtx, clientResp, c.callOptions...)
	}
	return serverResp, serverErr
}

func (c *connectNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := server.Context()

	clientResp, clientErr := c.client.Find(ctx, query, c.callOptions...)
	if clientErr != nil {
		return clientErr
	}

	for resp := range registry.ReadNetworkServiceChannel(clientResp) {
		if err := server.Send(resp); err != nil {
			return err
		}
	}

	return next.NetworkServiceRegistryServer(ctx).Find(query, server)
}

func (c *connectNSServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	_, clientErr := c.client.Unregister(ctx, in, c.callOptions...)
	_, serverErr := next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in)
	if clientErr != nil && serverErr != nil {
		return nil, errors.Wrapf(serverErr, "errors during client close: %v", clientErr)
	}
	if clientErr != nil {
		return nil, errors.Wrap(clientErr, "errors during client close")
	}
	return &empty.Empty{}, serverErr
}

// NewNetworkServiceRegistryServer - returns a connect chain element
func NewNetworkServiceRegistryServer(client registry.NetworkServiceRegistryClient, callOptions ...grpc.CallOption) registry.NetworkServiceRegistryServer {
	return &connectNSServer{
		client:      client,
		callOptions: callOptions,
	}
}
