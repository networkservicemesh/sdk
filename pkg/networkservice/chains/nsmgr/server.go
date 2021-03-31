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

	"github.com/google/uuid"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/filtermechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/roundrobin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	registryserialize "github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	registryadapter "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	registrychain "github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/nextwrap"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
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

type serverOptions struct {
	authorizeServer networkservice.NetworkServiceServer
	dialOptions     []grpc.DialOption
	regClientConn   *grpc.ClientConnInterface
	name            string
	url             string
}

// Option modifies server option value
type Option func(o *serverOptions)

// WithDialOptions sets gRPC Dial Options for the server
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = dialOptions
	}
}

// WithAuthorizeServer sets authorization server chain element
func WithAuthorizeServer(authorizeServer networkservice.NetworkServiceServer) Option {
	if authorizeServer == nil {
		panic("Authorize server cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeServer = authorizeServer
	}
}

// WithRegistryClientConn sets client connection to reach the upstream registry, if not passed memory storage will be used.
// Please do not pass nil value of registry connection.
func WithRegistryClientConn(regClientConn grpc.ClientConnInterface) Option {
	return func(o *serverOptions) {
		o.regClientConn = &regClientConn
	}
}

// WithName - set a nsmgr name, a default name is `Nsmgr`.
func WithName(name string) Option {
	return func(o *serverOptions) {
		o.name = name
	}
}

// WithURL - set a public URL address for NetworkServiceManager, it is used to access endpoints within this NSMgr from remote clients.
// Default value is not set, and endpoints address will not be changed.
func WithURL(url string) Option {
	return func(o *serverOptions) {
		o.url = url
	}
}

var _ Nsmgr = (*nsmgrServer)(nil)

// NewServer - Creates a new Nsmgr
//           tokenGenerator - authorization token generator
//			 options - a set of Nsmgr options.
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...Option) Nsmgr {
	opts := &serverOptions{
		authorizeServer: authorize.NewServer(authorize.Any()),
		name:            "nsmgr-" + uuid.New().String(),
		url:             "",
	}
	for _, opt := range options {
		opt(opts)
	}

	rv := &nsmgrServer{}

	var urlsRegistryServer, interposeRegistryServer registryapi.NetworkServiceEndpointRegistryServer

	var nsRegistry registryapi.NetworkServiceRegistryServer
	if opts.regClientConn != nil {
		// Use remote registry
		nsRegistry = registryadapter.NetworkServiceClientToServer(
			nextwrap.NewNetworkServiceRegistryClient(
				registryapi.NewNetworkServiceRegistryClient(*opts.regClientConn)))
	} else {
		// Use memory registry if no registry is passed
		nsRegistry = registrychain.NewNetworkServiceRegistryServer(
			registryserialize.NewNetworkServiceRegistryServer(),
			memory.NewNetworkServiceRegistryServer(),
		)
	}

	var nseRegistry registryapi.NetworkServiceEndpointRegistryServer
	if opts.regClientConn != nil {
		// Use remote registry
		nseRegistry = registryadapter.NetworkServiceEndpointClientToServer(
			nextwrap.NewNetworkServiceEndpointRegistryClient(
				registryapi.NewNetworkServiceEndpointRegistryClient(*opts.regClientConn)))
	} else {
		// Use memory registry if no registry is passed
		nseRegistry = registrychain.NewNetworkServiceEndpointRegistryServer(
			registryserialize.NewNetworkServiceEndpointRegistryServer(),
			memory.NewNetworkServiceEndpointRegistryServer(),
			setid.NewNetworkServiceEndpointRegistryServer(),
		)
	}

	localBypassRegistryServer := localbypass.NewNetworkServiceEndpointRegistryServer(opts.url)

	nseClient := next.NewNetworkServiceEndpointRegistryClient(
		registryserialize.NewNetworkServiceEndpointRegistryClient(),
		registryadapter.NetworkServiceEndpointServerToClient(localBypassRegistryServer),
		querycache.NewClient(ctx),
		registryadapter.NetworkServiceEndpointServerToClient(nseRegistry),
	)

	nsClient := registryadapter.NetworkServiceServerToClient(nsRegistry)

	// Construct Endpoint
	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAdditionalFunctionality(
			discover.NewServer(nsClient, nseClient),
			roundrobin.NewServer(),
			excludedprefixes.NewServer(ctx),
			recvfd.NewServer(), // Receive any files passed
			interpose.NewServer(&interposeRegistryServer),
			filtermechanisms.NewServer(&urlsRegistryServer),
			heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
			connect.NewServer(ctx,
				client.NewClientFactory(
					client.WithName(opts.name),
					client.WithAdditionalFunctionality(
						recvfd.NewClient(),
						sendfd.NewClient(),
					),
				),
				connect.WithDialOptions(opts.dialOptions...)),
			sendfd.NewServer()),
	)

	nsChain := registrychain.NewNamedNetworkServiceRegistryServer(opts.name+".NetworkServiceRegistry", nsRegistry)

	nseChain := registrychain.NewNamedNetworkServiceEndpointRegistryServer(
		opts.name+".NetworkServiceEndpointRegistry",
		registryserialize.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx, time.Minute),
		registryrecvfd.NewNetworkServiceEndpointRegistryServer(), // Allow to receive a passed files
		urlsRegistryServer,        // Store endpoints URLs
		interposeRegistryServer,   // Store cross connect NSEs
		localBypassRegistryServer, // Perform URL transformations
		nseRegistry,               // Register NSE inside Remote registry
	)
	rv.Registry = registry.NewServer(nsChain, nseChain)

	return rv
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
