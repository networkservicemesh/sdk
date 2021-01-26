// Copyright (c) 2020-2021 Cisco and/or its affiliates.
//
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

// Package nsmgr provides a Network Service Manager (nsmgrServer), but interface and implementation
package nsmgr

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	registry_recvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	chain_registry "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/nextwrap"

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
	networkservice.NetworkServiceServer
	networkservice.MonitorConnectionServer
	registry.Registry
}

type nsmgrServer struct {
	endpoint.Endpoint
	registry.Registry
}

var _ Nsmgr = (*nsmgrServer)(nil)

// NewServer - Creates a new Nsmgr
//           nsmRegistration - Nsmgr registration
//           authzServer - authorization server chain element
//           tokenGenerator - authorization token generator
//           registryCC - client connection to reach the upstream registry, could be nil, in this case only in memory storage will be used.
// 			 clientDialOptions -  a grpc.DialOption's to be passed to GRPC connections.
func NewServer(ctx context.Context, nsmRegistration *registryapi.NetworkServiceEndpoint, authzServer networkservice.NetworkServiceServer, tokenGenerator token.GeneratorFunc, registryCC grpc.ClientConnInterface, clientDialOptions ...grpc.DialOption) Nsmgr {
	rv := &nsmgrServer{}

	var urlsRegistryServer registryapi.NetworkServiceEndpointRegistryServer
	var localbypassRegistryServer registryapi.NetworkServiceEndpointRegistryServer

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

	nseClient := next.NewNetworkServiceEndpointRegistryClient(
		querycache.NewClient(ctx),
		adapter_registry.NetworkServiceEndpointServerToClient(nseRegistry))

	nsClient := adapter_registry.NetworkServiceServerToClient(nsRegistry)
	var interposeRegistry registryapi.NetworkServiceEndpointRegistryServer

	// Construct Endpoint
	rv.Endpoint = endpoint.NewServer(ctx,
		nsmRegistration.Name,
		authzServer,
		tokenGenerator,
		discover.NewServer(nsClient, nseClient),
		roundrobin.NewServer(),
		localbypass.NewServer(&localbypassRegistryServer),
		excludedprefixes.NewServer(ctx),
		recvfd.NewServer(), // Receive any files passed
		interpose.NewServer(&interposeRegistry),
		filtermechanisms.NewServer(&urlsRegistryServer),
		connect.NewServer(ctx,
			client.NewClientFactory(
				nsmRegistration.Name,
				addressof.NetworkServiceClient(adapters.NewServerToClient(rv)),
				tokenGenerator,
				recvfd.NewClient(),
				sendfd.NewClient(), // Send passed files.
			),
			clientDialOptions...),
		sendfd.NewServer(),
	)

	nsChain := chain_registry.NewNamedNetworkServiceRegistryServer(nsmRegistration.Name+".NetworkServiceRegistry", nsRegistry)

	nseChain := chain_registry.NewNamedNetworkServiceEndpointRegistryServer(
		nsmRegistration.Name+".NetworkServiceEndpointRegistry",
		expire.NewNetworkServiceEndpointRegistryServer(time.Minute),
		registry_recvfd.NewNetworkServiceEndpointRegistryServer(), // Allow to receive a passed files
		urlsRegistryServer,
		interposeRegistry,         // Store cross connect NSEs
		localbypassRegistryServer, // Store endpoint Id to EndpointURL for local access.
		seturl.NewNetworkServiceEndpointRegistryServer(nsmRegistration.Url), // Remember endpoint URL
		nseRegistry, // Register NSE inside Remote registry with ID assigned
	)
	rv.Registry = registry.NewServer(nsChain, nseChain)

	return rv
}

func newRemoteNSServer(cc grpc.ClientConnInterface) registryapi.NetworkServiceRegistryServer {
	if cc != nil {
		return adapter_registry.NetworkServiceClientToServer(
			nextwrap.NewNetworkServiceRegistryClient(
				registryapi.NewNetworkServiceRegistryClient(cc)))
	}
	return nil
}
func newRemoteNSEServer(cc grpc.ClientConnInterface) registryapi.NetworkServiceEndpointRegistryServer {
	if cc != nil {
		return adapter_registry.NetworkServiceEndpointClientToServer(
			nextwrap.NewNetworkServiceEndpointRegistryClient(
				registryapi.NewNetworkServiceEndpointRegistryClient(cc)))
	}
	return nil
}

func (n *nsmgrServer) Register(s *grpc.Server) {
	grpcutils.RegisterHealthServices(s, n, n.NetworkServiceEndpointRegistryServer(), n.NetworkServiceRegistryServer())
	networkservice.RegisterNetworkServiceServer(s, n)
	networkservice.RegisterMonitorConnectionServer(s, n)
	registryapi.RegisterNetworkServiceRegistryServer(s, n.Registry.NetworkServiceRegistryServer())
	registryapi.RegisterNetworkServiceEndpointRegistryServer(s, n.Registry.NetworkServiceEndpointRegistryServer())
}

var _ Nsmgr = &nsmgrServer{}
var _ endpoint.Endpoint = &nsmgrServer{}
var _ registry.Registry = &nsmgrServer{}
