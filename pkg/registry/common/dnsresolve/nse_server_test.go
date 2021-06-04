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

package dnsresolve_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type checkNSEContext struct {
	*testing.T
	expectedURL *url.URL
}

func (c *checkNSEContext) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	require.Equal(c, c.expectedURL, clienturlctx.ClientURL(ctx))
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, ns)
}

func (c *checkNSEContext) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	require.Equal(c, c.expectedURL, clienturlctx.ClientURL(s.Context()))
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func (c *checkNSEContext) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	require.Equal(c, c.expectedURL, clienturlctx.ClientURL(ctx))
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, ns)
}

func Test_DNSResolve(t *testing.T) {
	const srv = "service1"

	var resolver = new(sandbox.FakeDNSResolver)

	u, err := url.Parse("tcp://127.0.0.1:80")
	require.NoError(t, err)

	resolver.AddSRVEntry("domain1", srv, u)

	s := dnsresolve.NewNetworkServiceEndpointRegistryServer(
		dnsresolve.WithRegistryService(srv),
		dnsresolve.WithResolver(resolver),
	)

	s = next.NewNetworkServiceEndpointRegistryServer(s, &checkNSEContext{T: t, expectedURL: u})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"})
	require.NoError(t, err)
	err = s.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"}}, streamchannel.NewNetworkServiceEndpointFindServer((ctx), nil))
	require.NoError(t, err)
	_, err = s.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"})
	require.NoError(t, err)
}

func Test_DNSResolve_LookupNsmgrProxy(t *testing.T) {
	defer goleak.VerifyNone(t)

	const (
		regSrv        = "service1"
		nsmgrProxySrv = "service2"
		domain        = "domain1"
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var resolver = new(sandbox.FakeDNSResolver)

	regURL, err := url.Parse("tcp://127.0.0.1:80")
	require.NoError(t, err)
	resolver.AddSRVEntry(domain, regSrv, regURL)

	nsmgrProxyURL, err := url.Parse("tcp://127.0.0.1:81")
	require.NoError(t, err)
	resolver.AddSRVEntry(domain, nsmgrProxySrv, nsmgrProxyURL)

	s := dnsresolve.NewNetworkServiceEndpointRegistryServer(
		dnsresolve.WithRegistryService(regSrv),
		dnsresolve.WithNSMgrProxyService(nsmgrProxySrv),
		dnsresolve.WithResolver(resolver),
	)

	s = next.NewNetworkServiceEndpointRegistryServer(s, &checkNSEContext{T: t, expectedURL: regURL}, memory.NewNetworkServiceEndpointRegistryServer())

	_, err = s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1@" + domain})
	require.NoError(t, err)

	resp, err := adapters.NetworkServiceEndpointServerToClient(s).Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse-1@" + domain}})
	require.NoError(t, err)

	r, err := resp.Recv()
	require.NoError(t, err)
	require.Equal(t, nsmgrProxyURL.String(), r.Url)

	_, err = s.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1@" + domain})
	require.NoError(t, err)
}
