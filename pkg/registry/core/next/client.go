// Copyright (c) 2020 Cisco Systems, Inc.
//
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// NetworkServiceRegistryClientWrapper - a function that wraps around a registry.NetworkServiceRegistryClientWrapper
type NetworkServiceRegistryClientWrapper func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

// NetworkServiceRegistryClientChainer - a function that chains together a list of registry.NetworkServiceRegistryClientWrapper
type NetworkServiceRegistryClientChainer func(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

type nextNetworkServiceRegistryClient struct {
	clients    []registry.NetworkServiceRegistryClient
	index      int
	nextParent registry.NetworkServiceRegistryClient
}

func (n nextNetworkServiceRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryClient(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Register(withNextRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: n.nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Register(withNextRegistryClient(ctx, n.nextParent), in)
}

func (n nextNetworkServiceRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryClient(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Find(withNextRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: n.nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Find(withNextRegistryClient(ctx, n.nextParent), in)
}

func (n nextNetworkServiceRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*empty.Empty, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryClient(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].Unregister(withNextRegistryClient(ctx, &nextNetworkServiceRegistryClient{nextParent: n.nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].Unregister(withNextRegistryClient(ctx, n.nextParent), in)
}

// NewWrappedNetworkServiceRegistryClient chains together clients with wrapper wrapped around each one
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

// NewNetworkServiceRegistryClient - chains together clients into a single registry.NetworkServiceRegistryClientWrapper
func NewNetworkServiceRegistryClient(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return NewWrappedNetworkServiceRegistryClient(func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
		return client
	}, clients...)
}
