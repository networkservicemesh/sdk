// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

// Package proxydns provides default chain for stateless proxy registries based on DNS
package proxydns

import (
	"context"
	"net/url"

	"google.golang.org/grpc"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry"
	registryauthorize "github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/interdomainbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/replaceurl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/seturl"
	"github.com/networkservicemesh/sdk/pkg/registry/switchcase"

	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serverOptions struct {
	authorizeNSRegistryServer  registryapi.NetworkServiceRegistryServer
	authorizeNSERegistryServer registryapi.NetworkServiceEndpointRegistryServer
	authorizeNSRegistryClient  registryapi.NetworkServiceRegistryClient
	authorizeNSERegistryClient registryapi.NetworkServiceEndpointRegistryClient
	registryURL, nsmgrProxyURL *url.URL
	dialOptions                []grpc.DialOption
}

// Option modifies server option value
type Option func(o *serverOptions)

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
		panic("authorizeNSERegistryClient cannot be nil")
	}
	return func(o *serverOptions) {
		o.authorizeNSERegistryClient = authorizeNSERegistryClient
	}
}

// WithNSMgrProxyURL sets url to the nsmgr proxys
func WithNSMgrProxyURL(u *url.URL) Option {
	return func(o *serverOptions) {
		o.nsmgrProxyURL = u
	}
}

// WithRegistryURL sets url to the registry
func WithRegistryURL(u *url.URL) Option {
	return func(o *serverOptions) {
		o.registryURL = u
	}
}

// WithDialOptions sets grpc.DialOptions for the server
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = dialOptions
	}
}

// NewServer creates new stateless registry server that proxies queries to the second registries by DNS domains
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, dnsResolver dnsresolve.Resolver, options ...Option) registry.Registry {
	opts := &serverOptions{
		authorizeNSRegistryServer:  registryauthorize.NewNetworkServiceRegistryServer(registryauthorize.Any()),
		authorizeNSERegistryServer: registryauthorize.NewNetworkServiceEndpointRegistryServer(registryauthorize.Any()),
		authorizeNSRegistryClient:  registryauthorize.NewNetworkServiceRegistryClient(registryauthorize.Any()),
		authorizeNSERegistryClient: registryauthorize.NewNetworkServiceEndpointRegistryClient(registryauthorize.Any()),
	}
	for _, opt := range options {
		opt(opts)
	}

	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGenerator),
		opts.authorizeNSERegistryServer,
		begin.NewNetworkServiceEndpointRegistryServer(),
		interdomainbypass.NewNetworkServiceEndpointRegistryServer(),
		switchcase.NewNetworkServiceEndpointRegistryServer(
			switchcase.NSEServerCase{
				Condition: func(ctx context.Context, nse *registryapi.NetworkServiceEndpoint) bool {
					for _, service := range nse.GetNetworkServiceNames() {
						if interdomain.Is(service) {
							return true
						}
					}
					return interdomain.Is(nse.GetName())
				},
				Action: next.NewNetworkServiceEndpointRegistryServer(
					seturl.NewNetworkServiceEndpointRegistryServer(opts.nsmgrProxyURL),
					dnsresolve.NewNetworkServiceEndpointRegistryServer(dnsresolve.WithResolver(dnsResolver)),
				),
			},
			switchcase.NSEServerCase{
				Condition: switchcase.Otherwise[*registryapi.NetworkServiceEndpoint],
				Action: next.NewNetworkServiceEndpointRegistryServer(
					replaceurl.NewNetworkServiceEndpointRegistryServer(opts.nsmgrProxyURL),
					clienturl.NewNetworkServiceEndpointRegistryServer(opts.registryURL),
				),
			},
		),
		connect.NewNetworkServiceEndpointRegistryServer(
			chain.NewNetworkServiceEndpointRegistryClient(
				clientconn.NewNetworkServiceEndpointRegistryClient(),
				opts.authorizeNSERegistryClient,
				grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
				dial.NewNetworkServiceEndpointRegistryClient(ctx,
					dial.WithDialOptions(opts.dialOptions...),
				),
				connect.NewNetworkServiceEndpointRegistryClient(),
			),
		))
	nsChain := chain.NewNetworkServiceRegistryServer(
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGenerator),
		begin.NewNetworkServiceRegistryServer(),
		opts.authorizeNSRegistryServer,
		switchcase.NewNetworkServiceRegistryServer(
			switchcase.NSServerCase{
				Condition: func(ctx context.Context, ns *registryapi.NetworkService) bool {
					return interdomain.Is(ns.GetName())
				},
				Action: dnsresolve.NewNetworkServiceRegistryServer(dnsresolve.WithResolver(dnsResolver)),
			},
			switchcase.NSServerCase{
				Condition: switchcase.Otherwise[*registryapi.NetworkService],
				Action:    clienturl.NewNetworkServiceRegistryServer(opts.registryURL),
			},
		),
		connect.NewNetworkServiceRegistryServer(
			chain.NewNetworkServiceRegistryClient(
				clientconn.NewNetworkServiceRegistryClient(),
				opts.authorizeNSRegistryClient,
				grpcmetadata.NewNetworkServiceRegistryClient(),
				dial.NewNetworkServiceRegistryClient(
					ctx,
					dial.WithDialOptions(opts.dialOptions...),
				),
				connect.NewNetworkServiceRegistryClient(),
			),
		))
	return registry.NewServer(nsChain, nseChain)
}
