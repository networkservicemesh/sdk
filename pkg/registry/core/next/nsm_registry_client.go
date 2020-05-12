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

// NSMRegistryClientWrapper - a function that wraps around a registry.NSMRegistryClient
type NSMRegistryClientWrapper func(client registry.NsmRegistryClient) registry.NsmRegistryClient

// NSMRegistryClientChainer - a function that chains together a list of registry.NSMRegistryClient
type NSMRegistryClientChainer func(clients ...registry.NsmRegistryClient) registry.NsmRegistryClient

type nextNSMRegistryClient struct {
	clients    []registry.NsmRegistryClient
	index      int
	nextParent registry.NsmRegistryClient
}

func (n *nextNSMRegistryClient) RegisterNSM(ctx context.Context, in *registry.NetworkServiceManager, opts ...grpc.CallOption) (*registry.NetworkServiceManager, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NSMRegistryClient(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].RegisterNSM(withNSMRegistryClient(ctx, &nextNSMRegistryClient{nextParent: n.nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].RegisterNSM(withNSMRegistryClient(ctx, n.nextParent), in, opts...)
}

func (n *nextNSMRegistryClient) GetEndpoints(ctx context.Context, in *empty.Empty, opts ...grpc.CallOption) (*registry.NetworkServiceEndpointList, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NSMRegistryClient(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index].GetEndpoints(withNSMRegistryClient(ctx, &nextNSMRegistryClient{nextParent: n.nextParent, clients: n.clients, index: n.index + 1}), in, opts...)
	}
	return n.clients[n.index].GetEndpoints(withNSMRegistryClient(ctx, n.nextParent), in, opts...)
}

// NewWrappedNSMRegistryClient chains together clients with wrapper wrapped around each one
func NewWrappedNSMRegistryClient(wrapper NSMRegistryClientWrapper, clients ...registry.NsmRegistryClient) registry.NsmRegistryClient {
	rv := &nextNSMRegistryClient{clients: make([]registry.NsmRegistryClient, 0, len(clients))}
	for _, c := range clients {
		rv.clients = append(rv.clients, wrapper(c))
	}
	return rv
}

// NewNSMRegistryClient - chains together clients into a single registry.NSMRegistryClient
func NewNSMRegistryClient(clients ...registry.NsmRegistryClient) registry.NsmRegistryClient {
	return NewWrappedNSMRegistryClient(func(client registry.NsmRegistryClient) registry.NsmRegistryClient {
		return client
	}, clients...)
}
