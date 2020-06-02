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
)

// NSMRegistryServerWrapper - a function that wraps around a registry.NSMRegistryServer
type NSMRegistryServerWrapper func(client registry.NsmRegistryServer) registry.NsmRegistryServer

// NSMRegistryServerChainer - a function that chains together a list of registry.NSMRegistryServer
type NSMRegistryServerChainer func(clients ...registry.NsmRegistryServer) registry.NsmRegistryServer

type nextNSMRegistryServer struct {
	servers    []registry.NsmRegistryServer
	index      int
	nextParent registry.NsmRegistryServer
}

func (n *nextNSMRegistryServer) RegisterNSM(ctx context.Context, in *registry.NetworkServiceManager) (*registry.NetworkServiceManager, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NSMRegistryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].RegisterNSM(withNSMRegistryServer(ctx, &nextNSMRegistryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}), in)
	}
	return n.servers[n.index].RegisterNSM(withNSMRegistryServer(ctx, n.nextParent), in)
}

func (n *nextNSMRegistryServer) GetEndpoints(ctx context.Context, in *empty.Empty) (*registry.NetworkServiceEndpointList, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := NSMRegistryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].GetEndpoints(withNSMRegistryServer(ctx, &nextNSMRegistryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}), in)
	}
	return n.servers[n.index].GetEndpoints(withNSMRegistryServer(ctx, n.nextParent), in)
}

// NewWrappedNSMRegistryServer chains together clients with wrapper wrapped around each one
func NewWrappedNSMRegistryServer(wrapper NSMRegistryServerWrapper, clients ...registry.NsmRegistryServer) registry.NsmRegistryServer {
	rv := &nextNSMRegistryServer{servers: make([]registry.NsmRegistryServer, 0, len(clients))}
	for _, c := range clients {
		if c != nil {
			rv.servers = append(rv.servers, wrapper(c))
		}
	}
	return rv
}
