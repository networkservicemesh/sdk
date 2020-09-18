// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package memory

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	registry2 "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/proxy"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func NewServer(ctx context.Context, proxyRegistryURL *url.URL, options ...grpc.DialOption) registry2.Registry {
	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		setid.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
		proxy.NewNetworkServiceEndpointRegistryServer(proxyRegistryURL),
		connect.NewNetworkServiceEndpointRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient {
			return chain.NewNetworkServiceEndpointRegistryClient(
				registry.NewNetworkServiceEndpointRegistryClient(cc),
			)
		}, connect.WithClientDialOptions(options...)),
	)
	nsChain := chain.NewNetworkServiceRegistryServer(
		expire.NewNetworkServiceServer(adapters.NetworkServiceEndpointServerToClient(nseChain)),
		memory.NewNetworkServiceRegistryServer(),
		proxy.NewNetworkServiceRegistryServer(proxyRegistryURL),
		connect.NewNetworkServiceRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
			return chain.NewNetworkServiceRegistryClient(
				registry.NewNetworkServiceRegistryClient(cc),
			)
		}, connect.WithClientDialOptions(options...)),
	)

	return registry2.NewServer(nsChain, nseChain)
}
