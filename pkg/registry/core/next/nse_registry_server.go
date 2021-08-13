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

package next

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// NetworkServiceEndpointRegistryServerWrapper - function that wraps a registry server
type NetworkServiceEndpointRegistryServerWrapper func(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer

// NetworkServiceEndpointRegistryServerChainer - function that chains registry servers
type NetworkServiceEndpointRegistryServerChainer func(servers ...registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer

type nextNetworkServiceEndpointRegistryServer struct {
	servers    []registry.NetworkServiceEndpointRegistryServer
	index      int
	nextParent registry.NetworkServiceEndpointRegistryServer
}

// NewWrappedNetworkServiceEndpointRegistryServer - creates a chain of servers with each one wrapped in wrapper
func NewWrappedNetworkServiceEndpointRegistryServer(wrapper NetworkServiceEndpointRegistryServerWrapper, servers ...registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	rv := &nextNetworkServiceEndpointRegistryServer{servers: make([]registry.NetworkServiceEndpointRegistryServer, 0, len(servers))}
	for _, s := range servers {
		rv.servers = append(rv.servers, wrapper(s))
	}
	return rv
}

// NewNetworkServiceEndpointRegistryServer - creates a chain of servers
func NewNetworkServiceEndpointRegistryServer(servers ...registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return NewWrappedNetworkServiceEndpointRegistryServer(func(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
		return server
	}, servers...)
}

func (n *nextNetworkServiceEndpointRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	server, ctx := n.getServerAndContext(ctx)
	return server.Register(ctx, request)
}

func (n *nextNetworkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	server, ctx := n.getServerAndContext(s.Context())
	return server.Find(query, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s))
}

func (n *nextNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	server, ctx := n.getServerAndContext(ctx)
	return server.Unregister(ctx, request)
}

func (n *nextNetworkServiceEndpointRegistryServer) getServerAndContext(ctx context.Context) (registry.NetworkServiceEndpointRegistryServer, context.Context) {
	nextParent := n.nextParent
	if n.index == 0 {
		nextParent = NetworkServiceEndpointRegistryServer(ctx)
		if len(n.servers) == 0 {
			return nextParent, ctx
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index], withNextNSERegistryServer(ctx, &nextNetworkServiceEndpointRegistryServer{nextParent: nextParent, servers: n.servers, index: n.index + 1})
	}
	return n.servers[n.index], withNextNSERegistryServer(ctx, nextParent)
}
