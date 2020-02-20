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

// RegistryServerWrapper - function that wraps a registry server
type RegistryServerWrapper func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

// RegistryServerChainer - function that chains registry servers
type RegistryServerChainer func(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer

type nextRegistryServer struct {
	index   int
	servers []registry.NetworkServiceRegistryServer
}

// NewWrappedRegistryServer - creates a chain of servers with each one wrapped in wrapper
func NewWrappedRegistryServer(wrapper RegistryServerWrapper, servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	rv := &nextRegistryServer{
		servers: servers,
	}
	for i := range rv.servers {
		rv.servers[i] = wrapper(rv.servers[i])
	}
	return rv
}

// NewRegistryServer - creates a chain of servers
func NewRegistryServer(servers ...registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	if len(servers) == 0 {
		return &tailRegistryServer{}
	}
	return NewWrappedRegistryServer(notWrapServer, servers...)
}

func (n *nextRegistryServer) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, &nextRegistryServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RegisterNSE(withNextRegistryServer(ctx, nil), request)
}

func (n *nextRegistryServer) BulkRegisterNSE(server registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return n.servers[n.index].BulkRegisterNSE(server)
}

func (n *nextRegistryServer) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, &nextRegistryServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].RemoveNSE(withNextRegistryServer(ctx, nil), request)
}

func notWrapServer(c registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
	return c
}
