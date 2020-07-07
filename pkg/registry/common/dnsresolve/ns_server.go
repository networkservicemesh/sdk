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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

type dnsNSResolveServer struct {
	resolver      Resolver
	defaultDomain *Domain
}

// NewNetworkServiceRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl
func NewNetworkServiceRegistryServer(options ...Option) registry.NetworkServiceRegistryServer {
	r := &dnsNSResolveServer{
		resolver: net.DefaultResolver,
	}

	for _, o := range options {
		o.apply(r)
	}

	return r
}

func (d *dnsNSResolveServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	domain, err := domainOrDefault(ctx, d.defaultDomain)
	if err != nil {
		return nil, err
	}
	url, err := resolveDomain(ctx, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturl.WithClientURL(ctx, url)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (d *dnsNSResolveServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	ctx := s.Context()
	domain, err := domainOrDefault(ctx, d.defaultDomain)
	if err != nil {
		return err
	}
	url, err := resolveDomain(ctx, domain, d.resolver)
	if err != nil {
		return err
	}
	ctx = clienturl.WithClientURL(s.Context(), url)
	s = streamcontext.NetworkServiceRegistryFindServer(ctx, s)
	return next.NetworkServiceRegistryServer(s.Context()).Find(q, s)
}

func (d *dnsNSResolveServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	domain, err := domainOrDefault(ctx, d.defaultDomain)
	if err != nil {
		return nil, err
	}
	url, err := resolveDomain(ctx, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturl.WithClientURL(ctx, url)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func (d *dnsNSResolveServer) setResolver(r Resolver) {
	d.resolver = r
}

func (d *dnsNSResolveServer) setDomain(domain *Domain) {
	d.defaultDomain = domain
}

var _ Resolver = (*net.Resolver)(nil)
