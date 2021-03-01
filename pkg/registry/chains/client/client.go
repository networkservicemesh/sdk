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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/sendfd"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NewNetworkServiceEndpointRegistryClient creates a new NewNetworkServiceEndpointRegistryClient that can be used for NSE registration.
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, cc grpc.ClientConnInterface, additionalFunctionality ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return chain.NewNetworkServiceEndpointRegistryClient(
		append(
			append([]registry.NetworkServiceEndpointRegistryClient{
				refresh.NewNetworkServiceEndpointRegistryClient(refresh.WithChainContext(ctx)),
				sendfd.NewNetworkServiceEndpointRegistryClient(),
			}, additionalFunctionality...),
			registry.NewNetworkServiceEndpointRegistryClient(cc),
		)...,
	)
}

// NewNetworkServiceRegistryClient creates a new registry.NetworkServiceRegistryClient that can be used for registry.NetworkService registration. Can be used as for nse also for cross-nse goals.
func NewNetworkServiceRegistryClient(cc grpc.ClientConnInterface, additionalFunctionality ...registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return chain.NewNetworkServiceRegistryClient(
		append(
			additionalFunctionality,
			registry.NewNetworkServiceRegistryClient(cc),
		)...,
	)
}

// NewNetworkServiceEndpointRegistryInterposeClient creates a new registry.NetworkServiceEndpointRegistryClient that can be used for cross-nse registration
func NewNetworkServiceEndpointRegistryInterposeClient(ctx context.Context, cc grpc.ClientConnInterface, additionalFunctionality ...registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return chain.NewNetworkServiceEndpointRegistryClient(
		append(
			append([]registry.NetworkServiceEndpointRegistryClient{
				interpose.NewNetworkServiceEndpointRegistryClient(),
				refresh.NewNetworkServiceEndpointRegistryClient(refresh.WithChainContext(ctx)),
				sendfd.NewNetworkServiceEndpointRegistryClient(),
			}, additionalFunctionality...),
			registry.NewNetworkServiceEndpointRegistryClient(cc),
		)...,
	)
}
