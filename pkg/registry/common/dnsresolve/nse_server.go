// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022-2024 Cisco Systems, Inc.
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
	"fmt"
	"net"
	"net/url"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

type dnsNSEResolveServer struct {
	resolver                                                 Resolver
	registryService, registryProxyService, nsmgrProxyService string
}

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	var serverOptions = &options{
		resolver:             net.DefaultResolver,
		registryService:      DefaultRegistryService,
		registryProxyService: DefaultRegistryProxyService,
		nsmgrProxyService:    DefaultNSMgrProxyService,
	}

	for _, opt := range opts {
		opt(serverOptions)
	}

	r := &dnsNSEResolveServer{
		resolver:             serverOptions.resolver,
		registryService:      serverOptions.registryService,
		registryProxyService: serverOptions.registryProxyService,
		nsmgrProxyService:    serverOptions.nsmgrProxyService,
	}

	return r
}

func translateNSE(nse *registry.NetworkServiceEndpoint, translator func(string) string) {
	nse.Name = translator(nse.Name)

	for i := 0; i < len(nse.GetNetworkServiceNames()); i++ {
		var service = nse.GetNetworkServiceNames()[i]
		var target = translator(service)

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
}

// TODO: consider to return error if NSE is not consistent and have multi domains target.
func findDomainFromNSE(nse *registry.NetworkServiceEndpoint) string {
	var domain string

	for _, name := range append([]string{nse.Name}, nse.GetNetworkServiceNames()...) {
		if interdomain.Is(name) {
			domain = interdomain.Domain(name)
			break
		}
	}

	return domain
}

func (d *dnsNSEResolveServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	var domain = findDomainFromNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

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
	domain                        string
	externalDNSURL, nsmgrProxyURL *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *dnsFindNSEServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	translateNSE(nseResp.NetworkServiceEndpoint, func(str string) string {
		return interdomain.Join(str, s.domain)
	})

	if s.nsmgrProxyURL != nil {
		var u *url.URL
		var err error
		if s.externalDNSURL != nil {
			u, err = url.Parse(fmt.Sprintf("dns://%v/%v.%v:%v", s.externalDNSURL.Host, DefaultNSMgrProxyService, s.domain, s.nsmgrProxyURL.Port()))
		} else {
			u, err = url.Parse(fmt.Sprintf("dns://%v.%v:%v", DefaultNSMgrProxyService, s.domain, s.nsmgrProxyURL.Port()))
		}
		if err != nil {
			return err
		}
		nseResp.GetNetworkServiceEndpoint().Url = u.String()
	}

	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}

func (d *dnsNSEResolveServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	var ctx = s.Context()
	var domain = findDomainFromNSE(q.NetworkServiceEndpoint)
	var registryProxyURL, err = resolveDomain(ctx, d.registryProxyService, domain, d.resolver)
	if err != nil {
		registryProxyURL, err = resolveDomain(ctx, d.registryService, domain, d.resolver)
	}
	if err != nil {
		return err
	}
	var nsmgrProxyURL, _ = resolveDomain(ctx, d.nsmgrProxyService, domain, d.resolver)
	var externalDNSURL, _ = resolveDomain(ctx, DefaultExternalDNSService, domain, d.resolver)

	ctx = clienturlctx.WithClientURL(s.Context(), registryProxyURL)

	s = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)

	translateNSE(q.NetworkServiceEndpoint, interdomain.Target)

	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, &dnsFindNSEServer{NetworkServiceEndpointRegistry_FindServer: s, domain: domain, nsmgrProxyURL: nsmgrProxyURL, externalDNSURL: externalDNSURL})
}

func (d *dnsNSEResolveServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	var domain = findDomainFromNSE(nse)
	var u, err = resolveDomain(ctx, d.registryService, domain, d.resolver)

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

func (d *dnsNSEResolveServer) setRegistryService(service string) {
	d.registryService = service
}
