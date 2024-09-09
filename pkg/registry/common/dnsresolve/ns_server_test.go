// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type checkNSContext struct{ *testing.T }

func (c *checkNSContext) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	require.NotNil(c, clienturlctx.ClientURL(ctx))
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (c *checkNSContext) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	require.NotNil(c, clienturlctx.ClientURL(s.Context()))
	return next.NetworkServiceRegistryServer(s.Context()).Find(q, s)
}

func (c *checkNSContext) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	require.NotNil(c, clienturlctx.ClientURL(ctx))
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func TestDNSResolve_CorrectUsecase(t *testing.T) {
	const srv = "service1"

	resolver := sandbox.NewFakeResolver()

	u, err := url.Parse("tcp://127.0.0.1:80")
	require.NoError(t, err)

	require.NoError(t, sandbox.AddSRVEntry(resolver, "domain1", srv, u))

	s := dnsresolve.NewNetworkServiceRegistryServer(
		dnsresolve.WithRegistryService(srv),
		dnsresolve.WithResolver(resolver))

	s = next.NewNetworkServiceRegistryServer(s, &checkNSContext{t})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = s.Register(ctx, &registry.NetworkService{Name: "ns-1@domain1"})
	require.NoError(t, err)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: "ns-1@domain1"}}, streamchannel.NewNetworkServiceFindServer(ctx, nil))
	require.NoError(t, err)
	_, err = s.Unregister(ctx, &registry.NetworkService{Name: "ns-1@domain1"})
	require.NoError(t, err)
}

func TestDNSResolve_LoopUsecase(t *testing.T) {
	const srv = "service1"

	resolver := sandbox.NewFakeResolver()

	u, err := url.Parse("tcp://127.0.0.1:80")
	require.NoError(t, err)

	require.NoError(t, sandbox.AddSRVEntry(resolver, "domain1", srv, u))

	s := dnsresolve.NewNetworkServiceRegistryServer(
		dnsresolve.WithRegistryService(srv),
		dnsresolve.WithResolver(resolver),
	)

	s = next.NewNetworkServiceRegistryServer(s, &checkNSContext{t})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = s.Register(ctx, &registry.NetworkService{Name: "ns-1@domain1"})
	require.NoError(t, err)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: "ns-1@domain1"}}, streamchannel.NewNetworkServiceFindServer(ctx, nil))
	require.NoError(t, err)
	_, err = s.Unregister(ctx, &registry.NetworkService{Name: "ns-1@domain1"})
	require.NoError(t, err)
}
