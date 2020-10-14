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

package sandbox

import (
	"context"
	"net/url"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// SupplyNSMgrProxyFunc nsmgr proxy
type SupplyNSMgrProxyFunc func(context.Context, string, token.GeneratorFunc, ...grpc.DialOption) endpoint.Endpoint

// SupplyNSMgrFunc supplies NSMGR
type SupplyNSMgrFunc func(context.Context, *registryapi.NetworkServiceEndpoint, networkservice.NetworkServiceServer, token.GeneratorFunc, grpc.ClientConnInterface, ...grpc.DialOption) nsmgr.Nsmgr

// SupplyForwarderFunc supplies Forwarder
type SupplyForwarderFunc func(context.Context, string, token.GeneratorFunc, *url.URL, ...grpc.DialOption) endpoint.Endpoint

// SupplyRegistryFunc supplies Registry
type SupplyRegistryFunc func(ctx context.Context, proxyRegistryURL *url.URL, options ...grpc.DialOption) registry.Registry

// SupplyRegistryProxyFunc supplies registry proxy
type SupplyRegistryProxyFunc func(ctx context.Context, dnsResolver dnsresolve.Resolver, handlingDNSDomain string, proxyNSMgrURL *url.URL, options ...grpc.DialOption) registry.Registry

// Node is pair of Forwarder and NSMgr
type Node struct {
	Forwarder *EndpointEntry
	NSMgr     *NSMgrEntry
}

// RegistryEntry is pair of registry.Registry and url.URL
type RegistryEntry struct {
	registry.Registry
	URL *url.URL
}

// NSMgrEntry is pair of nsmgr.Nsmgr and url.URL
type NSMgrEntry struct {
	nsmgr.Nsmgr
	URL *url.URL
}

// EndpointEntry is pair of endpoint.Endpoint and url.URL
type EndpointEntry struct {
	endpoint.Endpoint
	URL *url.URL
}

// Domain contains attached to domain nodes, registry
type Domain struct {
	Nodes         []*Node
	NSMgrProxy    *EndpointEntry
	Registry      *RegistryEntry
	RegistryProxy *RegistryEntry
	DNSResolver   dnsresolve.Resolver
	Name          string
	resources     []context.CancelFunc
}

// Cleanup frees all resources related to the domain
func (d *Domain) Cleanup() {
	for _, r := range d.resources {
		r()
	}
}
