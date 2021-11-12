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

// Package memory provides registry chain based on memory chain elements
package memory

import (
	"context"
	"net/url"
	"time"

	"google.golang.org/grpc"

	registryserver "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/proxy"
	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setlogoption"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setpayload"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setregistrationtime"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NewServer creates new registry server based on memory storage
func NewServer(ctx context.Context, expiryDuration time.Duration, proxyRegistryURL *url.URL, dialOptions ...grpc.DialOption) registryserver.Registry {
	nseChain := chain.NewNetworkServiceEndpointRegistryServer(
		setlogoption.NewNetworkServiceEndpointRegistryServer(map[string]string{}),
		serialize.NewNetworkServiceEndpointRegistryServer(),
		setregistrationtime.NewNetworkServiceEndpointRegistryServer(),
		expire.NewNetworkServiceEndpointRegistryServer(ctx, expiryDuration),
		memory.NewNetworkServiceEndpointRegistryServer(),
		proxy.NewNetworkServiceEndpointRegistryServer(proxyRegistryURL),
		connect.NewNetworkServiceEndpointRegistryServer(ctx, connect.WithDialOptions(dialOptions...)),
	)
	nsChain := chain.NewNetworkServiceRegistryServer(
		setlogoption.NewNetworkServiceRegistryServer(map[string]string{}),
		serialize.NewNetworkServiceRegistryServer(),
		setpayload.NewNetworkServiceRegistryServer(),
		memory.NewNetworkServiceRegistryServer(),
		proxy.NewNetworkServiceRegistryServer(proxyRegistryURL),
		connect.NewNetworkServiceRegistryServer(ctx, connect.WithDialOptions(dialOptions...)),
	)

	return registryserver.NewServer(nsChain, nseChain)
}
