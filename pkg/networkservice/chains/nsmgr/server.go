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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	chain_registry "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/nextwrap"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"

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
	rv := &nsmgrServer{}

	var localbypassRegistryServer registry.NetworkServiceEndpointRegistryServer

	nsRegistry := newRemoteNSServer(registryCC)
	if nsRegistry == nil {
		// Use memory registry if no registry is passed
		nsRegistry = memory.NewNetworkServiceRegistryServer()
	}

	nseRegistry := newRemoteNSEServer(registryCC)
	if nseRegistry == nil {
		nseRegistry = chain_registry.NewNetworkServiceEndpointRegistryServer(
			setid.NewNetworkServiceEndpointRegistryServer(),  // If no remote registry then assign ID.
			memory.NewNetworkServiceEndpointRegistryServer(), // Memory registry to store result inside.
		)
	}

	// Construct Endpoint
	rv.Endpoint = endpoint.NewServer(
		nsmRegistration.Name,
		authzServer,
		tokenGenerator,
		discover.NewServer(adapter_registry.NetworkServiceServerToClient(nsRegistry), adapter_registry.NetworkServiceEndpointServerToClient(nseRegistry)),
		roundrobin.NewServer(),
		localbypass.NewServer(&localbypassRegistryServer),
		connect.NewServer(
			client.NewClientFactory(nsmRegistration.Name,
				addressof.NetworkServiceClient(
					adapters.NewServerToClient(rv)),
				tokenGenerator),
			clientDialOptions...),
	)

	rv.nsServer = chain_registry.NewNetworkServiceRegistryServer(nsRegistry)

	rv.nseServer = chain_registry.NewNetworkServiceEndpointRegistryServer(
		seturl.NewNetworkServiceEndpointRegistryServer(nsmRegistration.Url), // Remember endpoint URL
		nseRegistry,               // Register NSE inside Remote registry with ID assigned
		localbypassRegistryServer, // Store endpoint Id to EndpointURL for local access.
	)

	return rv
}

func newRemoteNSServer(cc grpc.ClientConnInterface) registry.NetworkServiceRegistryServer {
	if cc != nil {
		return adapter_registry.NetworkServiceClientToServer(
			nextwrap.NewNetworkServiceRegistryClient(
				registry.NewNetworkServiceRegistryClient(cc)))
	}
	return nil
}
func newRemoteNSEServer(cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryServer {
	if cc != nil {
		return adapter_registry.NetworkServiceEndpointClientToServer(
			nextwrap.NewNetworkServiceEndpointRegistryClient(
				registry.NewNetworkServiceEndpointRegistryClient(cc)))
	}
	return nil
}

func (n *nsmgrServer) Register(s *grpc.Server) {
	n.Endpoint.Register(s)

	// Register registry
	registry.RegisterNetworkServiceRegistryServer(s, n.NetworkServiceRegistryServer())
	registry.RegisterNetworkServiceEndpointRegistryServer(s, n.NetworkServiceEndpointRegistryServer())
}
