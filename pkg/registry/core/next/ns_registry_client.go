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

package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

// NetworkServiceRegistryClientWrapper - function that wraps a registry server
type NetworkServiceRegistryClientWrapper func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

// NetworkServiceRegistryClientChainer - function that chains registry servers
type NetworkServiceRegistryClientChainer func(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

type nextNetworkServiceRegistryClient struct {
	clients    []registry.NetworkServiceRegistryClient
	index      int
	nextParent registry.NetworkServiceRegistryClient
}

func (n *nextNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Find(withNextNSRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Find(withNextNSRegistryClient(ctx, nextParent), in)
}

func (n *nextNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Register(withNextNSRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Register(withNextNSRegistryClient(ctx, nextParent), in)
}

func (n *nextNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Unregister(withNextNSRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Unregister(withNextNSRegistryClient(ctx, nextParent), in)
}

// NewWrappedNetworkServiceRegistryClient - creates a chain of servers with each one wrapped in wrapper
func NewWrappedNetworkServiceRegistryClient(wrapper NetworkServiceRegistryClientWrapper, clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	if len(clients) == 0 {
		return &tailNetworkServiceRegistryClient{}
	}
	rv := &nextNetworkServiceRegistryClient{clients: make([]registry.NetworkServiceRegistryClient, 0, len(clients))}
	for _, c := range clients {
		rv.clients = append(rv.clients, wrapper(c))
	}
	return rv
}

// NewNetworkServiceRegistryClient - creates a chain of servers
func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return NewWrappedNetworkServiceRegistryClient(func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
		return client
	}, clients...)
}
