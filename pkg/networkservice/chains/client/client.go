// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

	"google.golang.org/grpc"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type clientOptions struct {
	name                    string
	onHeal                  *networkservice.NetworkServiceClient
	additionalFunctionality []networkservice.NetworkServiceClient
	authorizeClient         networkservice.NetworkServiceClient
}

// Option modifies default client chain values.
type Option func(c *clientOptions)

// WithHeal sets heal for the client.
func WithHeal(onHeal *networkservice.NetworkServiceClient) Option {
	return Option(func(c *clientOptions) {
		c.onHeal = onHeal
	})
}

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

// NewClient - returns a (1.) case NSM client.
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - cc - grpc.ClientConnInterface for the endpoint to which this client should connect
func NewClient(ctx context.Context, cc grpc.ClientConnInterface, clientOpts ...Option) networkservice.NetworkServiceClient {
	var rv networkservice.NetworkServiceClient
	var opts = &clientOptions{
		name:            "client-" + uuid.New().String(),
		authorizeClient: null.NewClient(),
		onHeal:          &rv,
	}
	for _, opt := range clientOpts {
		opt(opts)
	}
	rv = chain.NewNetworkServiceClient(
		append(
			append([]networkservice.NetworkServiceClient{
				updatepath.NewClient(opts.name),
				serialize.NewClient(),
				heal.NewClient(ctx, networkservice.NewMonitorConnectionClient(cc), opts.onHeal),
				refresh.NewClient(ctx),
				metadata.NewClient(),
			}, opts.additionalFunctionality...),
			opts.authorizeClient,
			networkservice.NewNetworkServiceClient(cc),
		)...)
	return rv
}

// Factory creates a networkservice.NetworkServiceClient by passed context.Cotnext and grpc.ClientConnInterface
type Factory = func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient

// NewCrossConnectClientFactory - returns a (2.) case func(cc grpc.ClientConnInterface) NSM client factory.
func NewCrossConnectClientFactory(clientOpts ...Option) Factory {
	return func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return chain.NewNetworkServiceClient(
			mechanismtranslation.NewClient(),
			NewClient(ctx, cc, clientOpts...),
		)
	}
}

// NewClientFactory - returns a (3.) case func(cc grpc.ClientConnInterface) NSM client factory.
func NewClientFactory(clientOpts ...Option) Factory {
	return func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
		return chain.NewNetworkServiceClient(
			NewClient(ctx, cc, clientOpts...),
		)
	}
}
