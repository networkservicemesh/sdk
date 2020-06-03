// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
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

package next

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nextNetworkServiceRegistryBulkRegisterNSEServer struct {
	registry.NetworkServiceRegistry_BulkRegisterNSEServer
	ctx context.Context
}

func (s *nextNetworkServiceRegistryBulkRegisterNSEServer) Context() context.Context {
	return s.ctx
}

// NetworkServiceRegistryServerWrapper - function that wraps a registry server
type NetworkServiceRegistryServerWrapper func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

// NetworkServiceRegistryServerChainer - function that chains registry servers
type NetworkServiceRegistryServerChainer func(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

type nextNetworkServiceRegistryServer struct {
	servers    []registry.NetworkServiceRegistryServer
	index      int
	nextParent registry.NetworkServiceRegistryServer
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

func (n *nextNetworkServiceRegistryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, &nextNetworkServiceRegistryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, n.nextParent), request)
}

func (n *nextNetworkServiceRegistryServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	ctx := server.Context()
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].BulkRegisterNSE(&nextNetworkServiceRegistryBulkRegisterNSEServer{
			NetworkServiceRegistry_BulkRegisterNSEServer: server,
			ctx: withNextRegistryServer(ctx, &nextNetworkServiceRegistryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}),
		})
	}
	return n.servers[n.index].BulkRegisterNSE(&nextNetworkServiceRegistryBulkRegisterNSEServer{
		NetworkServiceRegistry_BulkRegisterNSEServer: server,
		ctx: withNextRegistryServer(ctx, n.nextParent),
	})
}

func (n *nextNetworkServiceRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NetworkServiceRegistryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, &nextNetworkServiceRegistryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, n.nextParent), request)
}
