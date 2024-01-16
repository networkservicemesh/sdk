// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

type dnsNSResolveClient struct {
	resolver        Resolver
	registryService string
}

// NewNetworkServiceRegistryClient creates new NetworkServiceRegistryClient that can resolve passed domain to clienturl
func NewNetworkServiceRegistryClient(opts ...Option) registry.NetworkServiceRegistryClient {
	var clientOptions = &options{
		resolver:        net.DefaultResolver,
		registryService: DefaultRegistryService,
	}

	for _, opt := range opts {
		opt(clientOptions)
	}

	r := &dnsNSResolveClient{
		resolver:        clientOptions.resolver,
		registryService: clientOptions.registryService,
	}

	return r
}

func (d *dnsNSResolveClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	var original = ns.GetName()
	domain := interdomain.Domain(original)
	url, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, url)
	ns.Name = interdomain.Target(original)
	resp, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	resp.Name = original

	return resp, nil
}

type dnsNSResolveFindClient struct {
	registry.NetworkServiceRegistry_FindClient
	domain string
}

func (c *dnsNSResolveFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	resp, err := c.NetworkServiceRegistry_FindClient.Recv()
	if err != nil {
		return resp, err
	}
	resp.NetworkService.Name = interdomain.Join(resp.NetworkService.Name, c.domain)

	return resp, nil
}

func (d *dnsNSResolveClient) Find(ctx context.Context, q *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	domain := interdomain.Domain(q.NetworkService.Name)
	if domain == "" {
		return nil, errors.New("domain cannot be empty")
	}
	url, err := resolveDomain(ctx, d.registryService, domain, d.resolver)
	if err != nil {
		return nil, err
	}
	ctx = clienturlctx.WithClientURL(ctx, url)
	q.NetworkService.Name = interdomain.Target(q.NetworkService.Name)

	resp, err := next.NetworkServiceRegistryClient(ctx).Find(ctx, q, opts...)
	if err != nil {
		return nil, err
	}

	return &dnsNSResolveFindClient{
		NetworkServiceRegistry_FindClient: resp,
		domain:                            domain,
	}, nil
}

func (d *dnsNSResolveClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
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
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}
