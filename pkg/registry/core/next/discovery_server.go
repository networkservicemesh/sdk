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

	"github.com/networkservicemesh/api/pkg/api/registry"
)

// DiscoveryServerWrapper - a function that wraps around a registry.NetworkServiceDiscoveryServer
type DiscoveryServerWrapper func(server registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer

// DiscoveryServerChainer - a function that chains together a list of registry.NetworkServiceDiscoveryServer
type DiscoveryServerChainer func(servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer

type nextDiscoveryServer struct {
	servers    []registry.NetworkServiceDiscoveryServer
	index      int
	nextParent registry.NetworkServiceDiscoveryServer
}

// NewWrappedDiscoveryServer chains together servers with wrapper wrapped around each one
func NewWrappedDiscoveryServer(wrapper DiscoveryServerWrapper, servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	rv := &nextDiscoveryServer{servers: make([]registry.NetworkServiceDiscoveryServer, 0, len(servers))}
	for _, c := range servers {
		rv.servers = append(rv.servers, wrapper(c))
	}
	return rv
}

// NewDiscoveryServer - chains together servers into a single registry.NetworkServiceRegistryClientWrapper
func NewDiscoveryServer(servers ...registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
	return NewWrappedDiscoveryServer(func(server registry.NetworkServiceDiscoveryServer) registry.NetworkServiceDiscoveryServer {
		return server
	}, servers...)
}

func (n *nextDiscoveryServer) FindNetworkService(ctx context.Context, request *registry.FindNetworkServiceRequest) (*registry.FindNetworkServiceResponse, error) {
	if n.index == 0 && ctx != nil {
		if nextParent := DiscoveryServer(ctx); nextParent != nil {
			n.nextParent = nextParent
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].FindNetworkService(withNextDiscoveryServer(ctx, &nextDiscoveryServer{nextParent: n.nextParent, servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].FindNetworkService(withNextDiscoveryServer(ctx, n.nextParent), request)
}
