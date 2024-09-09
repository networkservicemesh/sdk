// Copyright (c) 2020-2021 Cisco Systems, Inc.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package adapters provide API to converting client to server and vise versa
package adapters

import (
	"context"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type networkServiceEndpointRegistryServer struct {
	client registry.NetworkServiceEndpointRegistryClient
}

func (n *networkServiceEndpointRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NewNetworkServiceEndpointRegistryClient(
		n.client,
		&callNextNSEServer{server: next.NetworkServiceEndpointRegistryServer(ctx)},
	).Register(ctx, request)
}

func (n *networkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	client, err := next.NewNetworkServiceEndpointRegistryClient(
		n.client,
		&callNextNSEServer{server: next.NetworkServiceEndpointRegistryServer(server.Context())},
	).Find(server.Context(), query)
	if client == nil || err != nil {
		return err
	}
	return nseFindClientToServer(client, server)
}

func (n *networkServiceEndpointRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NewNetworkServiceEndpointRegistryClient(
		n.client,
		&callNextNSEServer{server: next.NetworkServiceEndpointRegistryServer(ctx)},
	).Unregister(ctx, request)
}

// NetworkServiceEndpointClientToServer - returns a registry.NetworkServiceEndpointRegistryClient wrapped around the supplied client.
func NetworkServiceEndpointClientToServer(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryServer {
	return &networkServiceEndpointRegistryServer{client: client}
}

var _ registry.NetworkServiceEndpointRegistryServer = &networkServiceEndpointRegistryServer{}

type networkServiceEndpointRegistryClient struct {
	server registry.NetworkServiceEndpointRegistryServer
}

func (n *networkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NewNetworkServiceEndpointRegistryServer(
		n.server,
		&callNextNSEClient{client: next.NetworkServiceEndpointRegistryClient(ctx)},
	).Register(ctx, in)
}

func (n *networkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	s := next.NewNetworkServiceEndpointRegistryServer(
		n.server,
		&callNextNSEClient{client: next.NetworkServiceEndpointRegistryClient(ctx)},
	)
	return nseFindServerToClient(ctx, s, in)
}

func (n *networkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	return next.NewNetworkServiceEndpointRegistryServer(
		n.server,
		&callNextNSEClient{client: next.NetworkServiceEndpointRegistryClient(ctx)},
	).Unregister(ctx, in)
}

var _ registry.NetworkServiceEndpointRegistryClient = &networkServiceEndpointRegistryClient{}

// NetworkServiceEndpointServerToClient - returns a registry.NetworkServiceEndpointRegistryServer wrapped around the supplied server.
func NetworkServiceEndpointServerToClient(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryClient {
	return &networkServiceEndpointRegistryClient{server: server}
}

func nseFindClientToServer(client registry.NetworkServiceEndpointRegistry_FindClient, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	for {
		if err := client.Context().Err(); err != nil {
			break
		}
		msg, err := client.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return errors.Wrap(err, "NetworkServiceEndpointRegistry find client failed to get a message")
		}
		err = server.Send(msg)
		if err != nil {
			return errors.Wrapf(err, "NetworkServiceEndpointRegistry find server failed to send a message %s", msg)
		}
	}
	return nil
}

func nseFindServerToClient(ctx context.Context, server registry.NetworkServiceEndpointRegistryServer, in *registry.NetworkServiceEndpointQuery) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEndpointResponse, channelSize)
	s := streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch)
	if in != nil && in.GetWatch() {
		go func() {
			defer close(ch)
			_ = server.Find(in, s)
		}()
	} else {
		defer close(ch)
		if err := server.Find(in, s); err != nil {
			return nil, errors.Wrap(err, "NetworkServiceEndpointRegistry find server failed to find a query")
		}
	}
	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, ch), nil
}
