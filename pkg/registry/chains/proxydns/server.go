// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

	"google.golang.org/grpc"

	"github.com/google/uuid"

	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serverOptions struct {
	name        string
	dialOptions []grpc.DialOption
}

// Option modifies server option value
type Option func(o *serverOptions)

// WithName sets name for the registry memory server
func WithName(name string) Option {
	return Option(func(c *serverOptions) {
		c.name = name
	})
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
		name: "registry-proxy-" + uuid.New().String(),
	}
	for _, opt := range options {
		opt(opts)
	}

	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(opts.name),
		begin.NewNetworkServiceEndpointRegistryServer(),
		updatetoken.NewNetworkServiceEndpointRegistryServer(tokenGenerator),
		dnsresolve.NewNetworkServiceEndpointRegistryServer(dnsresolve.WithResolver(dnsResolver)),
		connect.NewNetworkServiceEndpointRegistryServer(
			chain.NewNetworkServiceEndpointRegistryClient(
				grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
				clientconn.NewNetworkServiceEndpointRegistryClient(),
				dial.NewNetworkServiceEndpointRegistryClient(ctx,
					dial.WithDialOptions(opts.dialOptions...),
				),
				connect.NewNetworkServiceEndpointRegistryClient(),
			),
		))
	nsChain := chain.NewNetworkServiceRegistryServer(
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(opts.name),
		begin.NewNetworkServiceRegistryServer(),
		updatetoken.NewNetworkServiceRegistryServer(tokenGenerator),
		dnsresolve.NewNetworkServiceRegistryServer(dnsresolve.WithResolver(dnsResolver)),
		connect.NewNetworkServiceRegistryServer(
			chain.NewNetworkServiceRegistryClient(
				grpcmetadata.NewNetworkServiceRegistryClient(),
				clientconn.NewNetworkServiceRegistryClient(),
				dial.NewNetworkServiceRegistryClient(
					ctx,
					dial.WithDialOptions(opts.dialOptions...),
				),
				connect.NewNetworkServiceRegistryClient(),
			),
		))
	return registry.NewServer(nsChain, nseChain)
}
