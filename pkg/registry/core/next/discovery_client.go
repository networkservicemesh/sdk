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

	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

// DiscoveryClientWrapper - a function that wraps around a registry.NetworkServiceDiscoveryClient
type DiscoveryClientWrapper func(client registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient

// DiscoveryClientChainer - a function that chains together a list of registry.NetworkServiceDiscoveryClient
type DiscoveryClientChainer func(clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient

type nextDiscoveryClient struct {
	index   int
	clients []registry.NetworkServiceDiscoveryClient
}

// NewWrappedDiscoveryClient chains together clients with wrapper wrapped around each one
func NewWrappedDiscoveryClient(wrapper DiscoveryClientWrapper, clients ...registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	rv := &nextDiscoveryClient{
		clients: clients,
	}
	for i := range rv.clients {
		rv.clients[i] = wrapper(rv.clients[i])
	}
	return rv
}

// NewDiscoveryClient - chains together clients into a single registry.NetworkServiceDiscoveryClient
func NewDiscoveryClient(clients []registry.NetworkServiceDiscoveryClient) registry.NetworkServiceDiscoveryClient {
	return NewWrappedDiscoveryClient(nil, clients...)
}

func (n *nextDiscoveryClient) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest, opts ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].FindNetworkService(withNextDiscoveryClient(ctx, &nextDiscoveryClient{clients: n.clients, index: n.index + 1}), request, opts...)
	}
	return n.clients[n.index].FindNetworkService(withNextDiscoveryClient(ctx, nil), request, opts...)
}
