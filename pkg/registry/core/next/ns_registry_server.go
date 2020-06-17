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

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// NetworkServiceRegistryServerWrapper - function that wraps a registry server
type NetworkServiceRegistryServerWrapper func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

// NetworkServiceRegistryServerChainer - function that chains registry servers
type NetworkServiceRegistryServerChainer func(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

type nextNetworkServiceRegistryServer struct {
	servers    []registry.NetworkServiceRegistryServer
	index      int
	nextParent registry.NetworkServiceRegistryServer
}

func (n *nextNetworkServiceRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextServer := NetworkServiceRegistryServer(ctx); nextServer != nil {
			nextParent = nextServer
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Register(withNextNSRegistryServer(ctx, &nextNetworkServiceRegistryServer{nextParent: nextParent, servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].Register(withNextNSRegistryServer(ctx, nextParent), request)
}

func (n *nextNetworkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	nextParent := n.nextParent
	if n.index == 0 && s.Context() != nil {
		if nextServer := NetworkServiceRegistryServer(s.Context()); nextServer != nil {
			nextParent = nextServer
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Find(query,
			streamcontext.NetworkServiceRegistryFindServer(
				withNextNSRegistryServer(
					s.Context(),
					&nextNetworkServiceRegistryServer{nextParent: nextParent, servers: n.servers, index: n.index + 1}),
				s,
			),
		)
	}
	return n.servers[n.index].Find(query,
		streamcontext.NetworkServiceRegistryFindServer(
			withNextNSRegistryServer(
				s.Context(),
				nextParent),
			s,
		),
	)
}

func (n *nextNetworkServiceRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	nextParent := n.nextParent
	if n.index == 0 && ctx != nil {
		if nextServer := NetworkServiceRegistryServer(ctx); nextServer != nil {
			nextParent = nextServer
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Unregister(withNextNSRegistryServer(ctx, &nextNetworkServiceRegistryServer{nextParent: nextParent, servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].Unregister(withNextNSRegistryServer(ctx, nextParent), request)
}

// NewWrappedNetworkServiceRegistryServer - creates a chain of servers with each one wrapped in wrapper
func NewWrappedNetworkServiceRegistryServer(wrapper NetworkServiceRegistryServerWrapper, servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	if len(servers) == 0 {
		return &tailNetworkServiceRegistryServer{}
	}
	rv := &nextNetworkServiceRegistryServer{servers: make([]registry.NetworkServiceRegistryServer, 0, len(servers))}
	for _, s := range servers {
		rv.servers = append(rv.servers, wrapper(s))
	}
	return rv
}

// NewNetworkServiceRegistryServer - creates a chain of servers
func NewNetworkServiceRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return NewWrappedNetworkServiceRegistryServer(func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
		return server
	}, servers...)
}
