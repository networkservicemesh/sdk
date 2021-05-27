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

package client

import (
	"context"
	"net/url"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NewNetworkServiceRegistryClient creates a new NewNetworkServiceRegistryClient that can be used for NS registration.
func NewNetworkServiceRegistryClient(ctx context.Context, connectTo *url.URL, opts ...Option) registry.NetworkServiceRegistryClient {
	clientOpts := new(clientOptions)
	for _, opt := range opts {
		opt(clientOpts)
	}

	var additionalFunctionality registry.NetworkServiceRegistryClient
	if len(clientOpts.nsAdditionalFunctionality) > 0 {
		additionalFunctionality = chain.NewNetworkServiceRegistryClient(additionalFunctionality)
	} else {
		additionalFunctionality = null.NewNetworkServiceRegistryClient()
	}

	return chain.NewNetworkServiceRegistryClient(
		serialize.NewNetworkServiceRegistryClient(),
		additionalFunctionality,
		adapters.NetworkServiceServerToClient(
			chain.NewNetworkServiceRegistryServer(
				clienturl.NewNetworkServiceRegistryServer(connectTo),
				connect.NewNetworkServiceRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
					return registry.NewNetworkServiceRegistryClient(cc)
				}, clientOpts.dialOptions...),
			),
		),
	)
}
