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

// Package memory provides registry chain based on memory chain elements
package memory

import (
	"context"
	"net/url"
	"time"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	registryserver "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect2"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setpayload"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setregistrationtime"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/switchcase"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

// NewServer creates new registry server based on memory storage
func NewServer(ctx context.Context, expiryDuration time.Duration, proxyRegistryURL *url.URL, dialOptions ...grpc.DialOption) registryserver.Registry {
	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		switchcase.NewNetworkServiceEndpointRegistryServer(switchcase.NSEServerCase{
			Condition: func(c context.Context, nse *registry.NetworkServiceEndpoint) bool {
				if interdomain.Is(nse.GetName()) {
					return true
				}
				for _, ns := range nse.GetNetworkServiceNames() {
					if interdomain.Is(ns) {
						return true
					}
				}
				return false
			},
			Action: chain.NewNetworkServiceEndpointRegistryServer(
				connect2.NewNetworkServiceEndpointRegistryServer(
					chain.NewNetworkServiceEndpointRegistryClient(
						begin.NewNetworkServiceEndpointRegistryClient(),
						clienturl.NewNetworkServiceEndpointRegistryClient(proxyRegistryURL),
						clientconn.NewNetworkServiceEndpointRegistryClient(),
						dial.NewNetworkServiceEndpointRegistryClient(ctx,
							dial.WithDialOptions(dialOptions...),
							dial.WithDialTimeout(time.Millisecond*100),
						),
						connect2.NewNetworkServiceEndpointRegistryClient(),
					),
				),
			),
		},
			switchcase.NSEServerCase{
				Condition: func(c context.Context, nse *registry.NetworkServiceEndpoint) bool { return true },
				Action: chain.NewNetworkServiceEndpointRegistryServer(
					setregistrationtime.NewNetworkServiceEndpointRegistryServer(),
					expire.NewNetworkServiceEndpointRegistryServer(ctx, expiryDuration),
					memory.NewNetworkServiceEndpointRegistryServer(),
				),
			},
		),
	)
	nsChain := chain.NewNetworkServiceRegistryServer(
		setpayload.NewNetworkServiceRegistryServer(),
		switchcase.NewNetworkServiceRegistryServer(
			switchcase.NSServerCase{
				Condition: func(c context.Context, ns *registry.NetworkService) bool {
					return interdomain.Is(ns.GetName())
				},
				Action: connect2.NewNetworkServiceRegistryServer(
					chain.NewNetworkServiceRegistryClient(
						clienturl.NewNetworkServiceRegistryClient(proxyRegistryURL),
						begin.NewNetworkServiceRegistryClient(),
						clientconn.NewNetworkServiceRegistryClient(),
						dial.NewNetworkServiceRegistryClient(ctx,
							dial.WithDialOptions(dialOptions...),
							dial.WithDialTimeout(time.Millisecond*100),
						),
						connect2.NewNetworkServiceRegistryClient(),
					),
				),
			},
			switchcase.NSServerCase{
				Condition: func(c context.Context, ns *registry.NetworkService) bool {
					return true
				},
				Action: memory.NewNetworkServiceRegistryServer(),
			},
		),
	)

	return registryserver.NewServer(nsChain, nseChain)
}
