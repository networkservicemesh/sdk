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

// Package registry provides a simple wrapper for building a Registry
package registry

import (
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

// Registry - aggregates the APIs:
//            - registry.NetworkServiceRegistryServer
//            - registry.NetworkServiceEndpointRegistryServer
type Registry interface {
	// NetworkServiceRegistryServer returns network service server
	NetworkServiceRegistryServer() registry.NetworkServiceRegistryServer
	// NetworkServiceRegistryServer returns network service endpoint server
	NetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer
	Register(s *grpc.Server)
}

type registryImpl struct {
	nsChain  registry.NetworkServiceRegistryServer
	nseChain registry.NetworkServiceEndpointRegistryServer
}

func (r *registryImpl) NetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return r.nsChain
}

func (r *registryImpl) NetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return r.nseChain
}

func (r *registryImpl) Register(server *grpc.Server) {
	grpcutils.RegisterHealthServices(server, r.nsChain, r.nseChain)
	registry.RegisterNetworkServiceRegistryServer(server, r.nsChain)
	registry.RegisterNetworkServiceEndpointRegistryServer(server, r.nseChain)
}

// NewServer creates new Registry with specific NetworkServiceRegistryServer and NetworkServiceEndpointRegistryServer functionality
func NewServer(nsChain registry.NetworkServiceRegistryServer, nseChain registry.NetworkServiceEndpointRegistryServer) Registry {
	return &registryImpl{
		nseChain: nseChain,
		nsChain:  nsChain,
	}
}
