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

// Package nsmgrproxy provides chain of networkservice.NetworkServiceServer chain elements to creating NSMgrProxy
package nsmgrproxy

import (
	"context"
	"net/url"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interdomainurl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/swapip"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	registryconnect "github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/proxy"
	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	registryadapter "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

func (n *nsmgrProxyServer) Register(s *grpc.Server) {
	grpcutils.RegisterHealthServices(s, n, n.NetworkServiceEndpointRegistryServer(), n.NetworkServiceRegistryServer())
	networkservice.RegisterNetworkServiceServer(s, n)
	networkservice.RegisterMonitorConnectionServer(s, n)
	registryapi.RegisterNetworkServiceRegistryServer(s, n.Registry.NetworkServiceRegistryServer())
	registryapi.RegisterNetworkServiceEndpointRegistryServer(s, n.Registry.NetworkServiceEndpointRegistryServer())
}

type nsmgrProxyServer struct {
	endpoint.Endpoint
	registry.Registry
}

type serverOptions struct {
	name                   string
	mapipFilePath          string
	listenOn               *url.URL
	authorizeServer        networkservice.NetworkServiceServer
	connectOptions         []connect.Option
	registryConnectOptions []registryconnect.Option
}

// Option modifies option value
type Option func(o *serverOptions)

// WithName sets name for the server
func WithName(name string) Option {
	return func(o *serverOptions) {
		o.name = name
	}
}

// WithAuthorizeServer sets authorize server for the server
func WithAuthorizeServer(authorizeServer networkservice.NetworkServiceServer) Option {
	if authorizeServer == nil {
		panic("Authorize server cannot be nil")
	}

	return func(o *serverOptions) {
		o.authorizeServer = authorizeServer
	}
}

// WithRegistryConnectOptions sets registry connect options
func WithRegistryConnectOptions(connectOptions ...registryconnect.Option) Option {
	return func(o *serverOptions) {
		o.registryConnectOptions = connectOptions
	}
}

// WithListenOn sets current listenOn url
func WithListenOn(u *url.URL) Option {
	return func(o *serverOptions) {
		o.listenOn = u
	}
}

// WithMapIPFilePath sets the custom path for the file that contains internal to external IPs information in YAML format
func WithMapIPFilePath(p string) Option {
	return func(o *serverOptions) {
		o.mapipFilePath = p
	}
}

// WithConnectOptions sets connect Options for the server
func WithConnectOptions(connectOptions ...connect.Option) Option {
	return func(o *serverOptions) {
		o.connectOptions = connectOptions
	}
}

// NewServer creates new proxy NSMgr
func NewServer(ctx context.Context, regURL, proxyURL *url.URL, tokenGenerator token.GeneratorFunc, options ...Option) nsmgr.Nsmgr {
	rv := new(nsmgrProxyServer)

	opts := &serverOptions{
		name:            "nsmgr-proxy-" + uuid.New().String(),
		authorizeServer: authorize.NewServer(authorize.Any()),
		listenOn:        &url.URL{Scheme: "unix", Host: "listen.on"},
		mapipFilePath:   "map-ip.yaml",
	}
	for _, opt := range options {
		opt(opts)
	}

	var nseStockServer registryapi.NetworkServiceEndpointRegistryServer

	nseClient := registryadapter.NetworkServiceEndpointServerToClient(
		chain.NewNetworkServiceEndpointRegistryServer(
			clienturl.NewNetworkServiceEndpointRegistryServer(regURL),
			registryconnect.NewNetworkServiceEndpointRegistryServer(ctx, opts.registryConnectOptions...),
		),
	)

	nsClient := registryadapter.NetworkServiceServerToClient(
		chain.NewNetworkServiceRegistryServer(
			clienturl.NewNetworkServiceRegistryServer(regURL),
			registryconnect.NewNetworkServiceRegistryServer(ctx, opts.registryConnectOptions...),
		),
	)

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAdditionalFunctionality(
			interdomainurl.NewServer(&nseStockServer),
			discover.NewServer(nsClient, nseClient),
			swapip.NewServer(ctx, opts.mapipFilePath),
			heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
			connect.NewServer(ctx,
				client.NewClientFactory(
					client.WithName(opts.name),
				), opts.connectOptions...,
			),
		),
	)

	var nsServerChain = chain.NewNetworkServiceRegistryServer(
		proxy.NewNetworkServiceRegistryServer(proxyURL),
		registryconnect.NewNetworkServiceRegistryServer(ctx, opts.registryConnectOptions...),
	)

	var nseServerChain = chain.NewNetworkServiceEndpointRegistryServer(
		proxy.NewNetworkServiceEndpointRegistryServer(proxyURL),
		seturl.NewNetworkServiceEndpointRegistryServer(opts.listenOn),
		nseStockServer,
		registryconnect.NewNetworkServiceEndpointRegistryServer(ctx, opts.registryConnectOptions...),
	)

	rv.Registry = registry.NewServer(nsServerChain, nseServerChain)
	return rv
}
