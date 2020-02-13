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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// RegistryClientWrapper - a function that wraps around a registry.NetworkServiceRegistryClient
type RegistryClientWrapper func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

// RegistryClientChainer - a function that chains together a list of registry.NetworkServiceRegistryClient
type RegistryClientChainer func(clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient

type nextRegistryClient struct {
	index   int
	clients []registry.NetworkServiceRegistryClient
}

// NewWrappedRegistryClient chains together clients with wrapper wrapped around each one
func NewWrappedRegistryClient(wrapper RegistryClientWrapper, clients ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	rv := &nextRegistryClient{
		clients: clients,
	}
	for i := range rv.clients {
		rv.clients[i] = wrapper(rv.clients[i])
	}
	return rv
}

// NewRegistryClient - chains together clients into a single registry.NetworkServiceRegistryClient
func NewRegistryClient(clients []registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return NewWrappedRegistryClient(notWrapClient, clients...)
}

func (n *nextRegistryClient) RegisterNSE(ctx context.Context, request *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].RegisterNSE(withNextRegistryClient(ctx, &nextRegistryClient{clients: n.clients, index: n.index + 1}), request)
	}
	return n.clients[n.index].RegisterNSE(withNextRegistryClient(ctx, &tailRegistryClient{}), request, opts...)
}

func (n *nextRegistryClient) BulkRegisterNSE(ctx context.Context, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].BulkRegisterNSE(withNextRegistryClient(ctx, &nextRegistryClient{clients: n.clients, index: n.index + 1}), opts...)
	}
	return n.clients[n.index].BulkRegisterNSE(withNextRegistryClient(ctx, &tailRegistryClient{}), opts...)
}

func (n *nextRegistryClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].RemoveNSE(withNextRegistryClient(ctx, &nextRegistryClient{clients: n.clients, index: n.index + 1}), request)
	}
	return n.clients[n.index].RemoveNSE(withNextRegistryClient(ctx, &tailRegistryClient{}), request, opts...)
}

func notWrapClient(c registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return c
}
