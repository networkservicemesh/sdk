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

// NetworkServiceEndpointRegistryClientWrapper - function that wraps a registry server
type NetworkServiceEndpointRegistryClientWrapper func(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient

// NetworkServiceEndpointRegistryClientChainer - function that chains registry servers
type NetworkServiceEndpointRegistryClientChainer func(clients ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient

type nextNetworkServiceEndpointRegistryClient struct {
	clients    []registry.NetworkServiceEndpointRegistryClient
	index      int
	nextParent registry.NetworkServiceEndpointRegistryClient
}

func (n *nextNetworkServiceEndpointRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceEndpointRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Find(withNextNSERegistryClient(ctx, &nextNetworkServiceEndpointRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Find(withNextNSERegistryClient(ctx, nextParent), in, opts...)
}

func (n *nextNetworkServiceEndpointRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceEndpointRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Register(withNextNSERegistryClient(ctx, &nextNetworkServiceEndpointRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Register(withNextNSERegistryClient(ctx, nextParent), in, opts...)
}

func (n *nextNetworkServiceEndpointRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextClient := NetworkServiceEndpointRegistryClient(ctx); nextClient != nil {
			nextParent = nextClient
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Unregister(withNextNSERegistryClient(ctx, &nextNetworkServiceEndpointRegistryClient{nextParent: nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Unregister(withNextNSERegistryClient(ctx, nextParent), in, opts...)
}

// NewWrappedNetworkServiceEndpointRegistryClient - creates a chain of servers with each one wrapped in wrapper
func NewWrappedNetworkServiceEndpointRegistryClient(wrapper NetworkServiceEndpointRegistryClientWrapper, clients ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	if len(clients) == 0 {
		return &tailNetworkServiceEndpointRegistryClient{}
	}
	rv := &nextNetworkServiceEndpointRegistryClient{clients: make([]registry.NetworkServiceEndpointRegistryClient, 0, len(clients))}
	for _, c := range clients {
		rv.clients = append(rv.clients, wrapper(c))
	}
	return rv
}

// NewNetworkServiceEndpointRegistryClient - creates a chain of servers
func NewNetworkServiceEndpointRegistryClient(clients ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return NewWrappedNetworkServiceEndpointRegistryClient(func(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
		return client
	}, clients...)
}
