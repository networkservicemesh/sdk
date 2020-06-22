// Copyright (c) 2020 Cisco Systems, Inc.
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
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type networkServiceEndpointRegistryServer struct {
	client registry.NetworkServiceEndpointRegistryClient
}

func (n *networkServiceEndpointRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return n.client.Register(ctx, request)
}

func (n *networkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	client, err := n.client.Find(s.Context(), query)
	if err != nil {
		return err
	}

	for {
		if err := client.Context().Err(); err != nil {
			return err
		}
		msg, err := client.Recv()
		if err != nil {
			if strings.HasSuffix(err.Error(), io.EOF.Error()) {
				return nil
			}
			return err
		}
		err = s.Send(msg)
		if err != nil {
			return err
		}
	}
}

func (n *networkServiceEndpointRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return n.client.Unregister(ctx, request)
}

// NetworkServiceEndpointClientToServer - returns a registry.NetworkServiceEndpointRegistryClient wrapped around the supplied client
func NetworkServiceEndpointClientToServer(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryServer {
	return &networkServiceEndpointRegistryServer{client: client}
}

var _ registry.NetworkServiceEndpointRegistryServer = &networkServiceEndpointRegistryServer{}

type networkServiceEndpointRegistryClient struct {
	server registry.NetworkServiceEndpointRegistryServer
}

func (n *networkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return n.server.Register(ctx, in)
}

func (n *networkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEndpoint, channelSize)
	s := streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch)
	err := n.server.Find(in, s)
	close(ch)
	if err != nil {
		return nil, err
	}
	return streamchannel.NewNetworkServiceEndpointFindClient(s.Context(), ch), nil
}

func (n *networkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return n.server.Unregister(ctx, in)
}

var _ registry.NetworkServiceEndpointRegistryClient = &networkServiceEndpointRegistryClient{}

// NetworkServiceEndpointServerToClient - returns a registry.NetworkServiceEndpointRegistryServer wrapped around the supplied server
func NetworkServiceEndpointServerToClient(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryClient {
	return &networkServiceEndpointRegistryClient{server: server}
}
