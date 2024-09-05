// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco Systems, Inc.
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

package proxydns_test

import (
	"context"
	"strings"
	"testing"
	"time"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

/*
TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
 1. local registry from domain2 has entry "ns-1"
 2. nsmgr from domain1 call find with query "ns-1@domain2"
 3. local registry proxies query to nsmgr proxy registry
 4. nsmgr proxy registry proxies query to proxy registry
 5. proxy registry proxies query to local registry from domain2

Expected: nsmgr found ns

domain1:                                                           domain2:
------------------------------------------------------------       -------------------
|                                                           | Find |                 |
|  local registry -> nsmgr proxy registry -> proxy registry | ---> | local registry  |
|                                                           |      |                 |
------------------------------------------------------------       ------------------.
*/
func TestInterdomainNetworkServiceRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := sandbox.NewFakeResolver()

	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()

	domain2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster.remote").
		Build()

	client1 := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain1.Registry.URL))

	client2 := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain2.Registry.URL))

	_, err := client2.Register(context.Background(), &registryapi.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	stream, err := client1.Find(ctx, &registryapi.NetworkServiceQuery{
		NetworkService: &registryapi.NetworkService{
			Name: "ns-1@" + domain2.Name,
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@"+domain2.Name, list[0].GetName())
}

/*
TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
 1. local registry from domain2 has entry "ns-1"
 2. nsmgr from domain1 call find with query "ns-1@domain2"
 3. local registry proxies query to nsmgr proxy registry
 4. nsmgr proxy registry proxies query to proxy registry
 5. proxy registry proxies query to local registry from domain2

Expected: nsmgr found ns

domain1:                                                             domain1						  domain1
------------------------------------------------------------         -------------------              ------------------------------------------------------------
|                                                           | 2.Find |                 | 1. Register  |                                                           |
|  local registry -> nsmgr proxy registry -> proxy registry |  --->  | local registry  | <----------- |  local registry -> nsmgr proxy registry -> proxy registry |
|                                                           |        |                 |              |                                                           |
------------------------------------------------------------         ------------------               ------------------------------------------------------------.
*/
func TestLocalDomain_NetworkServiceRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	dnsServer := sandbox.NewFakeResolver()

	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("cluster.local").
		SetDNSResolver(dnsServer).
		Build()

	client1 := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain1.Registry.URL))

	expected, err := client1.Register(context.Background(), &registryapi.NetworkService{
		Name: "ns-1@" + domain1.Name,
	})

	require.Nil(t, err)
	require.True(t, strings.Contains(expected.GetName(), "@"+domain1.Name))

	client2 := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain1.Registry.URL))

	stream, err := client2.Find(context.Background(), &registryapi.NetworkServiceQuery{
		NetworkService: &registryapi.NetworkService{
			Name: expected.GetName(),
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@cluster.local", list[0].GetName())
}

/*
	TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
		1. local registry from domain2 has entry "ns-1"
		2. nsmgr from domain1 call find with query "ns-1@domain2"
		3. local registry proxies query to nsmgr proxy registry
		4. nsmgr proxy registry proxies query to proxy registry
		5. proxy registry proxies query to local registry from domain2
	Expected: nsmgr found ns

	domain1:                                                             domain2:
	------------------------------------------------------------         -------------------              ------------------------------------------------------------
	|                                                           | 2.Find |                 | 1. Register  |                                                           |
	|  local registry -> nsmgr proxy registry -> proxy registry |  --->  | local registry  | <----------- |  local registry -> nsmgr proxy registry -> proxy registry |
	|                                                           |        |                 |              |                                                           |
	------------------------------------------------------------         ------------------               ------------------------------------------------------------
*/

func TestInterdomainFloatingNetworkServiceRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := sandbox.NewFakeResolver()

	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()

	domain2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()

	domain3 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("floating.domain").
		Build()

	registryClient := registryclient.NewNetworkServiceRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain2.Registry.URL))

	_, err := registryClient.Register(
		ctx,
		&registryapi.NetworkService{
			Name: "ns-1@" + domain3.Name,
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)

	defer func() {
		_ = cc.Close()
	}()

	client := registryapi.NewNetworkServiceRegistryClient(cc)

	stream, err := client.Find(ctx, &registryapi.NetworkServiceQuery{
		NetworkService: &registryapi.NetworkService{
			Name: "ns-1@" + domain3.Name,
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@"+domain3.Name, list[0].GetName())
}
