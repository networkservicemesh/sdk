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

// Package client provides a simple functions for building a NetworkServiceEndpointRegistryClient, NetworkServiceRegistryClient
package client

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connectto"
	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

// NewNetworkServiceEndpointRegistryClient creates a new NewNetworkServiceEndpointRegistryClient that can be used for NSE registration.
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, connectTo *url.URL, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	return newNetworkServiceEndpointRegistryClient(ctx, connectTo, false, opts...)
}

// NewNetworkServiceEndpointRegistryInterposeClient creates a new registry.NetworkServiceEndpointRegistryClient that can be used for cross-nse registration
func NewNetworkServiceEndpointRegistryInterposeClient(ctx context.Context, connectTo *url.URL, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	return newNetworkServiceEndpointRegistryClient(ctx, connectTo, true, opts...)
}

func newNetworkServiceEndpointRegistryClient(ctx context.Context, connectTo *url.URL, withInterpose bool, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	clientOpts := new(clientOptions)
	for _, opt := range opts {
		opt(clientOpts)
	}

	var interposeClient registry.NetworkServiceEndpointRegistryClient
	if withInterpose {
		interposeClient = interpose.NewNetworkServiceEndpointRegistryClient()
	} else {
		interposeClient = null.NewNetworkServiceEndpointRegistryClient()
	}

	c := new(registry.NetworkServiceEndpointRegistryClient)
	*c = chain.NewNetworkServiceEndpointRegistryClient(
		setid.NewNetworkServiceEndpointRegistryClient(),
		interposeClient,
		serialize.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		connectto.NewNetworkServiceEndpointRegistryClient(ctx, grpcutils.URLToTarget(connectTo),
			connectto.WithNSEAdditionalFunctionality(
				append(
					clientOpts.nseAdditionalFunctionality,
					sendfd.NewNetworkServiceEndpointRegistryClient())...),
			connectto.WithDialOptions(clientOpts.dialOptions...),
		),
	)

	return *c
}
