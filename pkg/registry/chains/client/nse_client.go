// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

// Package client provides a simple functions for building a NetworkServiceEndpointRegistryClient, NetworkServiceRegistryClient
package client

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/heal"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/retry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/metadata"
)

// NewNetworkServiceEndpointRegistryClient creates a new NewNetworkServiceEndpointRegistryClient that can be used for NSE registration.
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	clientOpts := &clientOptions{
		nseClientURLResolver:       null.NewNetworkServiceEndpointRegistryClient(),
		authorizeNSERegistryClient: authorize.NewNetworkServiceEndpointRegistryClient(authorize.Any()),
		dialTimeout:                time.Millisecond * 300,
	}
	for _, opt := range opts {
		opt(clientOpts)
	}

	return chain.NewNetworkServiceEndpointRegistryClient(
		append(
			[]registry.NetworkServiceEndpointRegistryClient{
				begin.NewNetworkServiceEndpointRegistryClient(),
				metadata.NewNetworkServiceEndpointClient(),
				querycache.NewNetworkServiceEndpointRegistryClient(ctx),
				retry.NewNetworkServiceEndpointRegistryClient(ctx),
				heal.NewNetworkServiceEndpointRegistryClient(ctx),
				refresh.NewNetworkServiceEndpointRegistryClient(ctx),
				clientOpts.authorizeNSERegistryClient,
				clientOpts.nseClientURLResolver,
				clientconn.NewNetworkServiceEndpointRegistryClient(),
				grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
				dial.NewNetworkServiceEndpointRegistryClient(ctx,
					dial.WithDialTimeout(clientOpts.dialTimeout),
					dial.WithDialOptions(clientOpts.dialOptions...),
				),
			},
			append(
				clientOpts.nseAdditionalFunctionality,
				connect.NewNetworkServiceEndpointRegistryClient(),
			)...,
		)...,
	)
}
