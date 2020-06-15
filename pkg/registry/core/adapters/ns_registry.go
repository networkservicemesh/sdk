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

package adapters

import (
	"context"
	"io"

	"google.golang.org/grpc"

	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type networkServiceRegistryServer struct {
	client registry.NetworkServiceRegistryClient
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	return n.client.Register(ctx, request)
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
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
			return err
		}
		err = s.Send(msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	return n.client.Unregister(ctx, request)
}

// NetworkServiceClientToServer - returns a registry.NetworkServiceRegistryClient wrapped around the supplied client
func NetworkServiceClientToServer(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	return &networkServiceRegistryServer{client: client}
}

var _ registry.NetworkServiceRegistryServer = &networkServiceRegistryServer{}

type networkServiceRegistryClient struct {
	server registry.NetworkServiceRegistryServer
}

func (n *networkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*registry.NetworkService, error) {
	return n.server.Register(ctx, in)
}

func (n networkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkService, channelSize)
	s := streamchannel.NewNetworkServiceFindServer(ctx, ch)
	if err := n.server.Find(in, s); err != nil {
		if err == io.EOF {
			close(ch)
		} else {
			return nil, err
		}
	}
	return streamchannel.NewNetworkServiceFindClient(s.Context(), ch), nil
}

func (n networkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return n.server.Unregister(ctx, in)
}

var _ registry.NetworkServiceRegistryClient = &networkServiceRegistryClient{}

// NetworkServiceServerToClient - returns a registry.NetworkServiceRegistryServer wrapped around the supplied server
func NetworkServiceServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &networkServiceRegistryClient{server: server}
}
