// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package nsmgr provides a Network Service Manager (nsmgrServer), but interface and implementation
package nsmgr

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	chain_registry "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	adapter_registry "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Nsmgr - A simple combintation of the Endpoint, registry.NetworkServiceRegistryServer, and registry.NetworkServiceDiscoveryServer interfaces
type Nsmgr interface {
	endpoint.Endpoint

	NetworkServiceRegistryServer() registry.NetworkServiceRegistryServer
	NetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer
}

type nsmgrServer struct {
	endpoint.Endpoint
	nsServer  registry.NetworkServiceRegistryServer
	nseServer registry.NetworkServiceEndpointRegistryServer

	serviceMap  memory.NetworkServiceSyncMap
	endpointMap memory.NetworkServiceEndpointSyncMap
}

func (n *nsmgrServer) NetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return n.nsServer
}

func (n *nsmgrServer) NetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return n.nseServer
}

// NewServer - Creates a new Nsmgr
//           nsmRegistration - Nsmgr registration
//           authzServer - authorization server chain element
//           tokenGenerator - authorization token generator
//           registryCC - client connection to reach the upstream registry, could be nil, in this case only in memory storage will be used.
// 			 clientDialOptions -  a grpc.DialOption's to be passed to GRPC connections.
func NewServer(nsmRegistration *registry.NetworkServiceEndpoint, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, registryCC grpc.ClientConnInterface, clientDialOptions ...grpc.DialOption) Nsmgr {
	// Construct callback server
	rv := &nsmgrServer{}

	var localbypassRegistryServer registry.NetworkServiceEndpointRegistryServer

	// Construct Endpoint
	rv.Endpoint = endpoint.NewServer(
		nsmRegistration.Name,
		authzServer,
		tokenGenerator,
		discover.NewServer(registry.NewNetworkServiceRegistryClient(registryCC), registry.NewNetworkServiceEndpointRegistryClient(registryCC)),
		roundrobin.NewServer(),
		localbypass.NewServer(&localbypassRegistryServer),
		connect.NewServer(
			client.NewClientFactory(nsmRegistration.Name,
				addressof.NetworkServiceClient(
					adapters.NewServerToClient(rv)),
				tokenGenerator),
			clientDialOptions...),
	)

	rv.nsServer = chain_registry.NewNetworkServiceRegistryServer(
		adapter_registry.NetworkServiceClientToServer(registry.NewNetworkServiceRegistryClient(registryCC)),
	)
	rv.nseServer = chain_registry.NewNetworkServiceEndpointRegistryServer(
		adapter_registry.NetworkServiceEndpointClientToServer(registry.NewNetworkServiceEndpointRegistryClient(registryCC)),
	)

	return rv
}

func (n *nsmgrServer) Register(s *grpc.Server) {
	n.Endpoint.Register(s)

	// Register registry
	registry.RegisterNetworkServiceRegistryServer(s, n.NetworkServiceRegistryServer())
	registry.RegisterNetworkServiceEndpointRegistryServer(s, n.NetworkServiceEndpointRegistryServer())
}
