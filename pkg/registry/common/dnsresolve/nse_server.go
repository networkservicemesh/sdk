// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2023 Cisco Systems, Inc.
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

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	serverOptions := &options{
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

func translateNSE(nse *registry.NetworkServiceEndpoint, translator func(string) string) {
	nse.Name = translator(nse.GetName())

	for i := 0; i < len(nse.GetNetworkServiceNames()); i++ {
		service := nse.GetNetworkServiceNames()[i]
		target := translator(service)

		nse.GetNetworkServiceNames()[i] = target

		if nse.GetNetworkServiceNames() == nil {
			continue
		}

		labels := nse.GetNetworkServiceLabels()[service]

		if labels == nil {
			continue
		}

		delete(nse.GetNetworkServiceLabels(), service)

		nse.GetNetworkServiceLabels()[target] = labels
	}
}

// TODO: consider to return error if NSE is not consistent and have multi domains target.
func resolveNSE(nse *registry.NetworkServiceEndpoint) string {
	var domain string

	for _, name := range append([]string{nse.GetName()}, nse.GetNetworkServiceNames()...) {
		if interdomain.Is(name) {
			domain = interdomain.Domain(name)
			break
		}
	}

	return domain
}

func (d *dnsNSEResolveServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	domain := resolveNSE(nse)
	u, err := resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	translateNSE(nse, interdomain.Target)

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	translateNSE(resp, func(s string) string {
		return interdomain.Join(s, domain)
	})

	return resp, nil
}

type dnsFindNSEServer struct {
	domain string
	nseURL *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *dnsFindNSEServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	translateNSE(nseResp.GetNetworkServiceEndpoint(), func(str string) string {
		return interdomain.Join(str, s.domain)
	})

	if s.nseURL != nil {
		nseResp.NetworkServiceEndpoint.Url = s.nseURL.String()
	}

	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}

func (d *dnsNSEResolveServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := s.Context()
	domain := resolveNSE(q.NetworkServiceEndpoint)
	nsmgrProxyURL, err := resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return err
	}

	ctx = clienturlctx.WithClientURL(s.Context(), nsmgrProxyURL)
	nsmgrProxyURL, err = resolveDomain(ctx, d.nsmgrProxyService, domain, d.resolver)
	if err != nil {
		log.FromContext(ctx).Errorf("nsmgrProxyService is not reachable by domain: %v", domain)
	}

	s = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)

	translateNSE(q.GetNetworkServiceEndpoint(), interdomain.Target)

	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, &dnsFindNSEServer{NetworkServiceEndpointRegistry_FindServer: s, domain: domain, nseURL: nsmgrProxyURL})
}

func (d *dnsNSEResolveServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	domain := resolveNSE(nse)
	u, err := resolveDomain(ctx, d.registryService, domain, d.resolver)

	if err != nil {
		return nil, err
	}

	ctx = clienturlctx.WithClientURL(ctx, u)

	translateNSE(nse, interdomain.Target)

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
