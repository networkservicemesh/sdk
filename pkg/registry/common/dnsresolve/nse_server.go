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
	"errors"
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

func (d *dnsNSEResolveServer) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	domain := interdomain.Domain(ns.Name)
	u, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, u)
	ns.Name = interdomain.Target(ns.Name)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, ns)
}

type dnsFindNSEServer struct {
	domain string
	u      *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *dnsFindNSEServer) Send(nse *registry.NetworkServiceEndpoint) error {
	nse.Name = interdomain.Join(nse.Name, s.domain)
	if s.u != nil {
		nse.Url = s.u.String()
	}
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nse)
}

func (d *dnsNSEResolveServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := s.Context()
	domain := domainOf(q.NetworkServiceEndpoint)
	if domain == "" {
		return errors.New("domain cannot be empty")
	}
	q.NetworkServiceEndpoint.Name = interdomain.Target(q.NetworkServiceEndpoint.Name)
	u, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return err
	}
	ctx = clienturlctx.WithClientURL(s.Context(), u)
	u, err = resolveDomain(ctx, d.nsmgrProxyService, domain, d.resolver)
	if err != nil {
		log.FromContext(ctx).Errorf("nsmgrProxyService is not reachable by domain: %v", domain)
	}
	s = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, &dnsFindNSEServer{NetworkServiceEndpointRegistry_FindServer: s, domain: domain, u: u})
}

func (d *dnsNSEResolveServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	domain := interdomain.Domain(ns.Name)
	u, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, u)
	ns.Name = interdomain.Target(ns.Name)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}

func domainOf(nse *registry.NetworkServiceEndpoint) string {
	if nse.Name != "" {
		return interdomain.Domain(nse.Name)
	}

	for i, ns := range nse.NetworkServiceNames {
		if interdomain.Is(ns) {
			result := interdomain.Domain(ns)
			nse.NetworkServiceNames[i] = interdomain.Target(ns)
			return result
		}
	}
	return ""
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
