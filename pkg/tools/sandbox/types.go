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

package sandbox

import (
	"context"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// SupplyNSMgrProxyFunc nsmgr proxy
type SupplyNSMgrProxyFunc func(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...nsmgrproxy.Option) endpoint.Endpoint

// SupplyNSMgrFunc supplies NSMGR
type SupplyNSMgrFunc func(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...nsmgr.Option) nsmgr.Nsmgr

// SupplyRegistryFunc supplies Registry
type SupplyRegistryFunc func(ctx context.Context, expiryDuration time.Duration, proxyRegistryURL *url.URL, options ...grpc.DialOption) registry.Registry

// SupplyRegistryProxyFunc supplies registry proxy
type SupplyRegistryProxyFunc func(ctx context.Context, dnsResolver dnsresolve.Resolver, handlingDNSDomain string, proxyNSMgrURL *url.URL, options ...grpc.DialOption) registry.Registry

// SetupNodeFunc setups each node on Builder.Build() stage
type SetupNodeFunc func(ctx context.Context, node *Node, i int)

type supplyURLFunc func(prefix string) *url.URL

// SupplyTokenGeneratorFunc supplies token generator func
type SupplyTokenGeneratorFunc func(timeout time.Duration) token.GeneratorFunc

// SupplyTransportCredentialsFunc supplies transport credentials
type SupplyTransportCredentialsFunc func() credentials.TransportCredentials

// RegistryEntry is pair of registry.Registry and url.URL
type RegistryEntry struct {
	registry.Registry
	URL *url.URL
}

// NSMgrEntry is pair of nsmgr.Nsmgr and url.URL
type NSMgrEntry struct {
	nsmgr.Nsmgr
	Name string
	URL  *url.URL
}

// EndpointEntry is pair of endpoint.Endpoint and url.URL
type EndpointEntry struct {
	endpoint.Endpoint
	Name string
	URL  *url.URL
}

// Domain contains attached to domain nodes, registry
type Domain struct {
	Nodes         []*Node
	NSMgrProxy    *EndpointEntry
	Registry      *RegistryEntry
	RegistryProxy *RegistryEntry

	supplyURL            supplyURLFunc
	supplyServerTC       SupplyTransportCredentialsFunc
	supplyClientTC       SupplyTransportCredentialsFunc
	supplyTokenGenerator SupplyTokenGeneratorFunc
}

// NodeConfig keeps custom node configuration parameters
type NodeConfig struct {
	NsmgrCtx                   context.Context
	NsmgrGenerateTokenFunc     token.GeneratorFunc
	ForwarderCtx               context.Context
	ForwarderGenerateTokenFunc token.GeneratorFunc
}
