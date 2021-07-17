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

package dnsresolve

import (
	"context"
	"net"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

type dnsNSEResolveServer struct {
	resolver          Resolver
	nsmgrProxyService string
	registryService   string
}

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	var serverOptions = &options{
		resolver:          net.DefaultResolver,
		registryService:   DefaultRegistryService,
		nsmgrProxyService: DefaultNsmgrProxyService,
	}

	for _, opt := range opts {
		opt(serverOptions)
	}

	r := &dnsNSEResolveServer{
		resolver:          serverOptions.resolver,
		nsmgrProxyService: serverOptions.nsmgrProxyService,
		registryService:   serverOptions.registryService,
	}

	return r
}

func resolveNSE(nse *registry.NetworkServiceEndpoint) string {
	var domain string

	for _, name := range append([]string{nse.Name}, nse.GetNetworkServiceNames()...) {
		if interdomain.Is(name) {
			domain = interdomain.Domain(name)
			break
		}
	}

	nse.Name = interdomain.Target(nse.Name)

	for i := 0; i < len(nse.GetNetworkServiceNames()); i++ {
		var service = nse.GetNetworkServiceNames()[i]
		var target = interdomain.Target(service)

		nse.GetNetworkServiceNames()[i] = target

		if nse.GetNetworkServiceNames() == nil {
			continue
		}

		var labels = nse.GetNetworkServiceLabels()[service]

		if labels == nil {
			continue
		}

		delete(nse.GetNetworkServiceLabels(), service)

		nse.GetNetworkServiceLabels()[target] = labels
	}

	return domain
}

func (d *dnsNSEResolveServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	var domain = resolveNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

type dnsFindNSEServer struct {
	domain string
	nseURL *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *dnsFindNSEServer) Send(nse *registry.NetworkServiceEndpoint) error {
	nse.Name = interdomain.Join(nse.Name, s.domain)

	if s.nseURL != nil {
		nse.Url = s.nseURL.String()
	}

	return s.NetworkServiceEndpointRegistry_FindServer.Send(nse)
}

func (d *dnsNSEResolveServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	var ctx = s.Context()
	var domain = resolveNSE(q.NetworkServiceEndpoint)
	var nsmgrProxyURL, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return err
	}

	ctx = clienturlctx.WithClientURL(s.Context(), nsmgrProxyURL)
	nsmgrProxyURL, err = resolveDomain(ctx, d.nsmgrProxyService, domain, d.resolver)

	if err != nil {
		log.FromContext(ctx).Errorf("nsmgrProxyService is not reachable by domain: %v", domain)
	}

	s = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)

	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, &dnsFindNSEServer{NetworkServiceEndpointRegistry_FindServer: s, domain: domain, nseURL: nsmgrProxyURL})
}

func (d *dnsNSEResolveServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	var domain = resolveNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func (d *dnsNSEResolveServer) setResolver(r Resolver) {
	d.resolver = r
}

func (d *dnsNSEResolveServer) setNSMgrProxyService(service string) {
	d.nsmgrProxyService = service
}

func (d *dnsNSEResolveServer) setRegistryService(service string) {
	d.registryService = service
}
