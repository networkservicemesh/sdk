// Copyright (c) 2020-2021 Cisco Systems, Inc.
//
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

// Package client provides a simple wrapper for building a NetworkServiceMeshClient
package client

import (
	"context"
	"net/url"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type clientOptions struct {
	name                    string
	additionalFunctionality []networkservice.NetworkServiceClient
	authorizeClient         networkservice.NetworkServiceClient
	dialOptions             []grpc.DialOption
	dialTimeout             time.Duration
}

// Option modifies default client chain values.
type Option func(c *clientOptions)

// WithName sets name for the client.
func WithName(name string) Option {
	return Option(func(c *clientOptions) {
		c.name = name
	})
}

// WithAdditionalFunctionality sets additionalFunctionality for the client. Note: this adds into tail of the client chain.
func WithAdditionalFunctionality(additionalFunctionality ...networkservice.NetworkServiceClient) Option {
	return Option(func(c *clientOptions) {
		c.additionalFunctionality = additionalFunctionality
	})
}

// WithAuthorizeClient sets authorizeClient for the client chain.
func WithAuthorizeClient(authorizeClient networkservice.NetworkServiceClient) Option {
	if authorizeClient == nil {
		panic("authorizeClient cannot be nil")
	}
	return Option(func(c *clientOptions) {
		c.authorizeClient = authorizeClient
	})
}

// WithDialOptions sets dial options
func WithDialOptions(dialOptions ...grpc.DialOption) Option {
	return Option(func(c *clientOptions) {
		c.dialOptions = dialOptions
	})
}

// WithDialTimeout sets dial timeout
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(c *clientOptions) {
		c.dialTimeout = dialTimeout
	}
}

// NewClient - returns a (1.) case NSM client.
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - cc - grpc.ClientConnInterface for the endpoint to which this client should connect
func NewClient(ctx context.Context, connectTo *url.URL, clientOpts ...Option) networkservice.NetworkServiceClient {
	rv := new(networkservice.NetworkServiceClient)
	var opts = &clientOptions{
		name:            "client-" + uuid.New().String(),
		authorizeClient: null.NewClient(),
		dialTimeout:     100 * time.Millisecond,
	}
	for _, opt := range clientOpts {
		opt(opts)
	}

	*rv = chain.NewNetworkServiceClient(
		updatepath.NewClient(opts.name),
		begin.NewClient(),
		metadata.NewClient(),
		refresh.NewClient(ctx),
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				heal.NewServer(ctx,
					heal.WithOnHeal(rv),
					heal.WithOnRestore(heal.OnRestoreRestore),
					heal.WithRestoreTimeout(time.Minute)),
				clienturl.NewServer(connectTo),
				connect.NewServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
					return chain.NewNetworkServiceClient(
						append(
							opts.additionalFunctionality,
							// TODO: move back to the end of the chain when `begin` chain element will be ready
							heal.NewClient(ctx, networkservice.NewMonitorConnectionClient(cc),
								heal.WithEndpointChange()),
							opts.authorizeClient,
							networkservice.NewNetworkServiceClient(cc),
						)...,
					)
				},
					connect.WithDialOptions(opts.dialOptions...),
					connect.WithDialTimeout(opts.dialTimeout)),
			),
		),
	)
	return *rv
}

// NewClientFactory - returns a (3.) case func(cc grpc.ClientConnInterface) NSM client factory.
func NewClientFactory(clientOpts ...Option) connect.ClientFactory {
	return func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		var rv networkservice.NetworkServiceClient
		var opts = &clientOptions{
			name:            "client-" + uuid.New().String(),
			authorizeClient: null.NewClient(),
		}
		for _, opt := range clientOpts {
			opt(opts)
		}
		rv = chain.NewNetworkServiceClient(
			append(
				append([]networkservice.NetworkServiceClient{
					updatepath.NewClient(opts.name),
					begin.NewClient(),
					metadata.NewClient(),
					refresh.NewClient(ctx),
					// TODO: move back to the end of the chain when `begin` chain element will be ready
					heal.NewClient(ctx, networkservice.NewMonitorConnectionClient(cc)),
				}, opts.additionalFunctionality...),
				opts.authorizeClient,
				networkservice.NewNetworkServiceClient(cc),
			)...)
		return rv
	}
}
