// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco Systems, Inc.
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

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
)

type dnsNSResolveServer struct {
	resolver        Resolver
	registryService string
}

// NewNetworkServiceRegistryServer creates new NetworkServiceRegistryServer that can resolve passed domain to clienturl
func NewNetworkServiceRegistryServer(opts ...Option) registry.NetworkServiceRegistryServer {
	var serverOptions = &options{
		resolver:        net.DefaultResolver,
		registryService: DefaultRegistryService,
	}

	for _, opt := range opts {
		opt(serverOptions)
	}

	r := &dnsNSResolveServer{
		resolver:        serverOptions.resolver,
		registryService: serverOptions.registryService,
	}

	return r
}

func (d *dnsNSResolveServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	domain := interdomain.Domain(ns.Name)
	url, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, url)
	ns.Name = interdomain.Target(ns.Name)
	resp, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}

	resp.Name = interdomain.Join(resp.Name, domain)

	return resp, nil
}

type dnsFindNSServer struct {
	domain string
	registry.NetworkServiceRegistry_FindServer
}

func (s *dnsFindNSServer) Send(nseResp *registry.NetworkServiceResponse) error {
	nseResp.NetworkService.Name = interdomain.Join(nseResp.NetworkService.Name, s.domain)
	return s.NetworkServiceRegistry_FindServer.Send(nseResp)
}

func (d *dnsNSResolveServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	ctx := s.Context()
	domain := interdomain.Domain(q.NetworkService.Name)
	if domain == "" {
		return errors.New("domain cannot be empty")
	}
	url, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return err
	}
	ctx = clienturlctx.WithClientURL(s.Context(), url)
	s = streamcontext.NetworkServiceRegistryFindServer(ctx, s)
	q.NetworkService.Name = interdomain.Target(q.NetworkService.Name)
	return next.NetworkServiceRegistryServer(s.Context()).Find(q, &dnsFindNSServer{domain: domain, NetworkServiceRegistry_FindServer: s})
}

func (d *dnsNSResolveServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	domain := interdomain.Domain(ns.Name)
	url, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, url)
	ns.Name = interdomain.Target(ns.Name)
	defer func() {
		ns.Name = interdomain.Join(ns.Name, domain)
	}()
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
