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

	"google.golang.org/grpc"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/externalips"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/interdomainurl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/swapip"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serverOptions struct {
	name            string
	authorizeServer networkservice.NetworkServiceServer
	dialOptions     []grpc.DialOption
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

// WithDialOptions sets gRPC Dial Options for the server
func WithDialOptions(options ...grpc.DialOption) Option {
	return func(o *serverOptions) {
		o.dialOptions = options
	}
}

// NewServer creates new proxy NSMgr
func NewServer(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...Option) endpoint.Endpoint {
	type nsmgrProxyServer struct {
		endpoint.Endpoint
	}
	rv := nsmgrProxyServer{}

	opts := &serverOptions{
		name:            "nsmgr-proxy-" + uuid.New().String(),
		authorizeServer: authorize.NewServer(authorize.Any()),
	}
	for _, opt := range options {
		opt(opts)
	}

	rv.Endpoint = endpoint.NewServer(ctx, tokenGenerator,
		endpoint.WithName(opts.name),
		endpoint.WithAuthorizeServer(opts.authorizeServer),
		endpoint.WithAdditionalFunctionality(
			interdomainurl.NewServer(),
			externalips.NewServer(ctx),
			swapip.NewServer(),
			heal.NewServer(ctx, addressof.NetworkServiceClient(adapters.NewServerToClient(rv))),
			connect.NewServer(ctx,
				client.NewClientFactory(client.WithName(opts.name)),
				connect.WithDialOptions(opts.dialOptions...),
			),
		),
	)
	return rv
}
