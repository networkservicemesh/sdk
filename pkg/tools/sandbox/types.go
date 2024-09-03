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

package sandbox

import (
	"context"
	"net/url"
	"time"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/registry"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/chains/proxydns"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// SupplyNSMgrProxyFunc nsmgr proxy.
type SupplyNSMgrProxyFunc func(ctx context.Context, regURL, proxyURL *url.URL, tokenGenerator token.GeneratorFunc, options ...nsmgrproxy.Option) nsmgr.Nsmgr

// SupplyNSMgrFunc supplies NSMGR.
type SupplyNSMgrFunc func(ctx context.Context, tokenGenerator token.GeneratorFunc, options ...nsmgr.Option) nsmgr.Nsmgr

// SupplyRegistryFunc supplies Registry.
type SupplyRegistryFunc func(ctx context.Context, tokenGenerator token.GeneratorFunc, defaultExpiration time.Duration, proxyRegistryURL *url.URL, options ...grpc.DialOption) registry.Registry

// SupplyRegistryProxyFunc supplies registry proxy.
type SupplyRegistryProxyFunc func(ctx context.Context, tokenGenerator token.GeneratorFunc, dnsResolver dnsresolve.Resolver, options ...proxydns.Option) registry.Registry

// SetupNodeFunc setups each node on Builder.Build() stage.
type SetupNodeFunc func(ctx context.Context, node *Node, nodeNum int)

// RegistryEntry is pair of registry.Registry and url.URL.
type RegistryEntry struct {
	URL *url.URL

	*restartableServer
	registry.Registry
}

// NSMgrEntry is pair of nsmgr.Nsmgr and url.URL.
type NSMgrEntry struct {
	Name string
	URL  *url.URL

	*restartableServer
	nsmgr.Nsmgr
}

// EndpointEntry is pair of endpoint.Endpoint and url.URL.
type EndpointEntry struct {
	Name string
	URL  *url.URL

	*restartableServer
	endpoint.Endpoint
	registryapi.NetworkServiceEndpointRegistryClient
}

// Domain contains attached to domain nodes, registry.
type Domain struct {
	Nodes         []*Node
	NSMgrProxy    *NSMgrEntry
	Registry      *RegistryEntry
	RegistryProxy *RegistryEntry

	DNSResolver dnsresolve.Resolver
	Name        string

	supplyURL func(prefix string) *url.URL
}

// NewNSRegistryClient creates new NS registry client for the domain.
func (d *Domain) NewNSRegistryClient(ctx context.Context, generatorFunc token.GeneratorFunc, opts ...registryclient.Option) registryapi.NetworkServiceRegistryClient {
	var registryURL *url.URL
	switch {
	case d.Registry != nil:
		registryURL = CloneURL(d.Registry.URL)
	case len(d.Nodes) != 0:
		registryURL = CloneURL(d.Nodes[0].NSMgr.URL)
	default:
		return nil
	}

	opts = append(opts,
		registryclient.WithClientURL(registryURL),
		registryclient.WithDialOptions(DialOptions(WithTokenGenerator(generatorFunc))...))

	return registryclient.NewNetworkServiceRegistryClient(ctx, opts...)
}
