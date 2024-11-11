// Copyright (c) 2020-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2020-2024 Doc.ai and/or its affiliates.
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
	"net/url"
	"time"

	"github.com/google/uuid"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discoverforwarder"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/metrics"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/netsvcmonitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	registryclientinfo "github.com/networkservicemesh/sdk/pkg/registry/common/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	registryconnect "github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	registryrecvfd "github.com/networkservicemesh/sdk/pkg/registry/common/recvfd"
	registrysendfd "github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"

	registryadapter "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	authmonitor "github.com/networkservicemesh/sdk/pkg/tools/monitorconnection/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Nsmgr - A simple combination of the Endpoint, registry.NetworkServiceRegistryServer, and registry.NetworkServiceDiscoveryServer interfaces
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
	authorizeServer                  networkservice.NetworkServiceServer
	authorizeMonitorConnectionServer networkservice.MonitorConnectionServer
	authorizeNSRegistryServer        registryapi.NetworkServiceRegistryServer
	authorizeNSRegistryClient        registryapi.NetworkServiceRegistryClient
	authorizeNSERegistryServer       registryapi.NetworkServiceEndpointRegistryServer
	authorizeNSERegistryClient       registryapi.NetworkServiceEndpointRegistryClient
	defaultExpiration                time.Duration
	dialOptions                      []grpc.DialOption
	dialTimeout                      time.Duration
	regURL                           *url.URL
	name                             string
	url                              string
	forwarderServiceName             string
}

// Option modifies server option value
type Option func(o *serverOptions)

// WithDialOptions sets grpc.DialOptions for the client
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = dialOptions
	}
}

// WithForwarderServiceName overrides default forwarder service name
// By default "forwarder"
func WithForwarderServiceName(forwarderServiceName string) Option {
	return func(o *serverOptions) {
		o.forwarderServiceName = forwarderServiceName
	}
}

// WithDefaultExpiration sets the default expiration for endpoints
func WithDefaultExpiration(d time.Duration) Option {
	return func(o *serverOptions) {
		o.defaultExpiration = d
	}
}

// WithDialTimeout sets dial timeout for the client
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *serverOptions) {
		o.dialTimeout = dialTimeout
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

// WithAuthorizeMonitorConnectionServer sets authorization MonitorConnectionServer chain element
func WithAuthorizeMonitorConnectionServer(authorizeMonitorConnectionServer networkservice.MonitorConnectionServer) Option {
	if authorizeMonitorConnectionServer == nil {
		panic("authorizeMonitorConnectionServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeMonitorConnectionServer = authorizeMonitorConnectionServer
	}
}

// WithAuthorizeNSRegistryServer sets authorization NetworkServiceRegistry chain element
func WithAuthorizeNSRegistryServer(authorizeNSRegistryServer registryapi.NetworkServiceRegistryServer) Option {
	if authorizeNSRegistryServer == nil {
		panic("authorizeNSRegistryServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSRegistryServer = authorizeNSRegistryServer
	}
}

// WithAuthorizeNSERegistryServer sets authorization NetworkServiceEndpointRegistry chain element
func WithAuthorizeNSERegistryServer(authorizeNSERegistryServer registryapi.NetworkServiceEndpointRegistryServer) Option {
	if authorizeNSERegistryServer == nil {
		panic("authorizeNSERegistryServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSERegistryServer = authorizeNSERegistryServer
	}
}

// WithAuthorizeNSRegistryClient sets authorization NetworkServiceRegistry chain element
func WithAuthorizeNSRegistryClient(authorizeNSRegistryClient registryapi.NetworkServiceRegistryClient) Option {
	if authorizeNSRegistryClient == nil {
		panic("authorizeNSRegistryClient cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSRegistryClient = authorizeNSRegistryClient
	}
}

// WithAuthorizeNSERegistryClient sets authorization NetworkServiceEndpointRegistry chain element
func WithAuthorizeNSERegistryClient(authorizeNSERegistryClient registryapi.NetworkServiceEndpointRegistryClient) Option {
	if authorizeNSERegistryClient == nil {
		panic("authorizeNSERegistryServer cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSERegistryClient = authorizeNSERegistryClient
	}
}

// WithRegistry sets URL and dial options to reach the upstream registry, if not passed memory storage will be used.
func WithRegistry(regURL *url.URL) Option {
	return func(o *serverOptions) {
		o.regURL = regURL
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
func WithURL(u string) Option {
	return func(o *serverOptions) {
		o.url = u
	}
}

var _ Nsmgr = (*nsmgrServer)(nil)

// NewServer - Creates a new Nsmgr
//
//	          tokenGenerator - authorization token generator
//				 options - a set of Nsmgr options.
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...Option) Nsmgr {
	opts := &serverOptions{
		authorizeServer:                  authorize.NewServer(authorize.Any()),
		authorizeMonitorConnectionServer: authmonitor.NewMonitorConnectionServer(authmonitor.Any()),
		authorizeNSRegistryServer:        registryauthorize.NewNetworkServiceRegistryServer(registryauthorize.Any()),
		authorizeNSRegistryClient:        registryauthorize.NewNetworkServiceRegistryClient(registryauthorize.Any()),
		authorizeNSERegistryServer:       registryauthorize.NewNetworkServiceEndpointRegistryServer(registryauthorize.Any()),
		authorizeNSERegistryClient:       registryauthorize.NewNetworkServiceEndpointRegistryClient(registryauthorize.Any()),
		defaultExpiration:                time.Minute,
		dialTimeout:                      time.Millisecond * 300,
		name:                             "nsmgr-" + uuid.New().String(),
		forwarderServiceName:             "forwarder",
	}
	for _, opt := range options {
		opt(opts)
	}

	rv := &nsmgrServer{}
	var nsRegistry = memory.NewNetworkServiceRegistryServer()
	if opts.regURL != nil {
		// Use remote registry
		nsRegistry = registryconnect.NewNetworkServiceRegistryServer(
			chain.NewNetworkServiceRegistryClient(
				clienturl.NewNetworkServiceRegistryClient(opts.regURL),
				begin.NewNetworkServiceRegistryClient(),
				querycache.NewNetworkServiceRegistryClient(ctx),
				clientconn.NewNetworkServiceRegistryClient(),
				opts.authorizeNSRegistryClient,
				grpcmetadata.NewNetworkServiceRegistryClient(),
				dial.NewNetworkServiceRegistryClient(ctx,
					dial.WithDialTimeout(opts.dialTimeout),
					dial.WithDialOptions(opts.dialOptions...),
				),
				registryconnect.NewNetworkServiceRegistryClient(),
			),
		)
	}

	nsRegistry = chain.NewNetworkServiceRegistryServer(
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGenerator),
		opts.authorizeNSRegistryServer,
		nsRegistry,
	)

	var remoteOrLocalRegistry = chain.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(opts.url),
		registryconnect.NewNetworkServiceEndpointRegistryServer(
			chain.NewNetworkServiceEndpointRegistryClient(
				begin.NewNetworkServiceEndpointRegistryClient(),
				querycache.NewNetworkServiceEndpointRegistryClient(ctx),
				clienturl.NewNetworkServiceEndpointRegistryClient(opts.regURL),
				clientconn.NewNetworkServiceEndpointRegistryClient(),
				opts.authorizeNSERegistryClient,
				grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
				dial.NewNetworkServiceEndpointRegistryClient(ctx,
					dial.WithDialTimeout(opts.dialTimeout),
					dial.WithDialOptions(opts.dialOptions...),
				),
				registryconnect.NewNetworkServiceEndpointRegistryClient(),
			),
		),
	)

	if opts.regURL == nil {
		remoteOrLocalRegistry = chain.NewNetworkServiceEndpointRegistryServer(
			memory.NewNetworkServiceEndpointRegistryServer(),
			localbypass.NewNetworkServiceEndpointRegistryServer(opts.url),
		)
	}

	var nseRegistry = chain.NewNetworkServiceEndpointRegistryServer(
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGenerator),
		opts.authorizeNSERegistryServer,
		begin.NewNetworkServiceEndpointRegistryServer(),
		registryclientinfo.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx, expire.WithDefaultExpiration(opts.defaultExpiration)),
		registryrecvfd.NewNetworkServiceEndpointRegistryServer(), // Allow to receive a passed files
		registrysendfd.NewNetworkServiceEndpointRegistryServer(),
		remoteOrLocalRegistry,
	)
	// Construct Endpoint
	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAuthorizeMonitorConnectionServer(opts.authorizeMonitorConnectionServer),
		endpoint.WithAdditionalFunctionality(
			adapters.NewClientToServer(clientinfo.NewClient()),
			discoverforwarder.NewServer(
				registryadapter.NetworkServiceServerToClient(nsRegistry),
				registryadapter.NetworkServiceEndpointServerToClient(remoteOrLocalRegistry),
				discoverforwarder.WithForwarderServiceName(opts.forwarderServiceName),
				discoverforwarder.WithNSMgrURL(opts.url),
			),
			netsvcmonitor.NewServer(ctx,
				registryadapter.NetworkServiceServerToClient(nsRegistry),
				registryadapter.NetworkServiceEndpointServerToClient(remoteOrLocalRegistry),
			),
			excludedprefixes.NewServer(ctx),
			recvfd.NewServer(), // Receive any files passed
			metrics.NewServer(),
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithName(opts.name),
					client.WithAdditionalFunctionality(
						recvfd.NewClient(),
						sendfd.NewClient(),
					),
					client.WithDialOptions(opts.dialOptions...),
					client.WithDialTimeout(opts.dialTimeout),
					client.WithoutRefresh(),
				),
			),
			sendfd.NewServer()),
	)

	rv.Registry = registry.NewServer(
		nsRegistry,
		nseRegistry,
	)
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
