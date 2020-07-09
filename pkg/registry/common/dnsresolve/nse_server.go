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

package dnsresolve

import (
	"context"
	"net"

	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

type dnsNSEResolveServer struct {
	resolver Resolver
	service  string
}

func (d *dnsNSEResolveServer) setService(service string) {
	d.service = service
}

// NewNetworkServiceEndpointRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl
func NewNetworkServiceEndpointRegistryServer(options ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &dnsNSEResolveServer{
		resolver: net.DefaultResolver,
		service:  NSMRegistryService,
	}

	for _, o := range options {
		o.apply(r)
	}

	return r
}

func (d *dnsNSEResolveServer) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	domain := interdomain.Domain(ns.Name)
	url, err := resolveDomain(ctx, d.service, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturl.WithClientURL(ctx, url)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, ns)
}

func (d *dnsNSEResolveServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := s.Context()
	domain := interdomain.Domain(q.NetworkServiceEndpoint.Name)
	url, err := resolveDomain(ctx, d.service, domain, d.resolver)
	if err != nil {
		return err
	}
	ctx = clienturl.WithClientURL(s.Context(), url)
	s = streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, s)
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func (d *dnsNSEResolveServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	domain := interdomain.Domain(ns.Name)
	url, err := resolveDomain(ctx, d.service, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturl.WithClientURL(ctx, url)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}

func (d *dnsNSEResolveServer) setResolver(r Resolver) {
	d.resolver = r
}
