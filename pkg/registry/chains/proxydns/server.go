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

// Package proxydns provides default chain for stateless proxy registries based on DNS
package proxydns

import (
	"context"
	"net/url"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/swap"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

type serverOptions struct {
	dnsResolver       dnsresolve.Resolver
	handlingDNSDomain string
	dialOptions       []grpc.DialOption
}

// Option modifies default server chain values
type Option func(o *serverOptions)

// WithDNSResolver sets resolver for the server
func WithDNSResolver(resolver dnsresolve.Resolver) Option {
	return func(o *serverOptions) {
		o.dnsResolver = resolver
	}
}

// WithHandlingDNSDomain sets handling domain name for the server
func WithHandlingDNSDomain(domain string) Option {
	return func(o *serverOptions) {
		o.handlingDNSDomain = domain
	}
}

// WithDialOptions sets gRPC Dial Options for the server
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = opts
	}
}

// NewServer creates new stateless registry server that proxies queries to the second registries by DNS domains
func NewServer(ctx context.Context, proxyNSMgrURL *url.URL, options ...Option) registry.Registry {
	opts := &serverOptions{}

	for _, opt := range options {
		opt(opts)
	}

	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		dnsresolve.NewNetworkServiceEndpointRegistryServer(dnsresolve.WithResolver(opts.dnsResolver)),
		swap.NewNetworkServiceEndpointRegistryServer(opts.handlingDNSDomain, proxyNSMgrURL),
		connect.NewNetworkServiceEndpointRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registryapi.NetworkServiceEndpointRegistryClient {
			return registryapi.NewNetworkServiceEndpointRegistryClient(cc)
		}, connect.WithClientDialOptions(opts.dialOptions...)))

	nsChain := chain.NewNetworkServiceRegistryServer(
		dnsresolve.NewNetworkServiceRegistryServer(dnsresolve.WithResolver(opts.dnsResolver)),
		swap.NewNetworkServiceRegistryServer(opts.handlingDNSDomain),
		connect.NewNetworkServiceRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registryapi.NetworkServiceRegistryClient {
			return chain.NewNetworkServiceRegistryClient(registryapi.NewNetworkServiceRegistryClient(cc))
		}, connect.WithClientDialOptions(opts.dialOptions...)))

	return registry.NewServer(nsChain, nseChain)
}
