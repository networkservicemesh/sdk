// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package forwarder provides utility method for building interpose endpoint
package forwarder

import (
	"context"
	"net/url"

	"github.com/google/uuid"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serverOptions struct {
	name                                      string
	authorizeServer                           networkservice.NetworkServiceServer
	beforeConnectServers, afterConnectServers []networkservice.NetworkServiceServer
	connectClients                            []networkservice.NetworkServiceClient
	connectOptions                            []connect.Option
}

// Option modifies server option value
type Option func(o *serverOptions)

// WithName sets name of the NetworkServiceServer
func WithName(name string) Option {
	return func(o *serverOptions) {
		o.name = name
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

// WithBeforeConnectServers sets before connect server chain element
func WithBeforeConnectServers(beforeConnectServers ...networkservice.NetworkServiceServer) Option {
	return func(o *serverOptions) {
		o.beforeConnectServers = beforeConnectServers
	}
}

// WithAfterConnectServers sets after connect server chain element
func WithAfterConnectServers(afterConnectServers ...networkservice.NetworkServiceServer) Option {
	return func(o *serverOptions) {
		o.afterConnectServers = afterConnectServers
	}
}

// WithConnectClients sets connect client chain element
func WithConnectClients(connectClients ...networkservice.NetworkServiceClient) Option {
	return func(o *serverOptions) {
		o.connectClients = connectClients
	}
}

// WithConnectOptions sets connect Options for the server
func WithConnectOptions(connectOptions ...connect.Option) Option {
	return func(o *serverOptions) {
		o.connectOptions = connectOptions
	}
}

// NewServer returns a new interpose endpoint chain
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, clientURL *url.URL, options ...Option) endpoint.Endpoint {
	opts := &serverOptions{
		name:            "forwarder-" + uuid.New().String(),
		authorizeServer: authorize.NewServer(authorize.Any()),
	}
	for _, opt := range options {
		opt(opts)
	}

	rv := new(struct{ endpoint.Endpoint })
	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAdditionalFunctionality(
			chain.NewNetworkServiceServer(opts.beforeConnectServers...),
			clienturl.NewServer(clientURL),
			heal.NewServer(ctx,
				heal.WithOnHeal(addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
				heal.WithOnRestore(heal.OnRestoreIgnore)),
			connect.NewServer(ctx,
				client.NewClientFactory(
					client.WithName(opts.name),
					client.WithAdditionalFunctionality(
						mechanismtranslation.NewClient(),
						chain.NewNetworkServiceClient(opts.connectClients...),
					),
				),
				opts.connectOptions...,
			),
			chain.NewNetworkServiceServer(opts.afterConnectServers...),
		),
	)

	return rv
}
