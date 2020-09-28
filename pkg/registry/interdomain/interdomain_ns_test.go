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

package interdomain_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	registry2 "github.com/networkservicemesh/sdk/pkg/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

/*
	TestInterdomainNetworkServiceRegistry covers the next scenario:
		1. local registry from domain2 has entry "ns-1"
		2. nsmgr from domain1 call find with query "ns-1@domain2"
		3. local registry proxies query to proxy registry
		4. proxy registry proxies query to local registry from domain2
	Expected: nsmgr found ns
	domain1                                      domain2
	 ___________________________________         ___________________
	|                                   | Find  |                   |
	| local registry --> proxy registry | ----> | local registry    |
	|                                   |       |                   |
	____________________________________         ___________________
*/
func TestInterdomainNetworkServiceRegistry(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.local.registry.proxy"
	const remoteRegistryDomain = "domain2.local.registry"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := new(sandbox.FakeDNSResolver)

	domain1 := sandbox.NewBuilder(t).
		SetContext(ctx).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()
	defer domain1.Cleanup()

	domain2 := sandbox.NewBuilder(t).
		SetContext(ctx).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()
	defer domain2.Cleanup()

	require.NoError(t, dnsServer.Register(localRegistryDomain, domain1.Registry.URL))
	require.NoError(t, dnsServer.Register(proxyRegistryDomain, domain1.RegistryProxy.URL))
	require.NoError(t, dnsServer.Register(remoteRegistryDomain, domain2.Registry.URL))

	_, err := domain2.Registry.NetworkServiceRegistryServer().Register(
		context.Background(),
		&registry.NetworkService{
			Name: "ns-1",
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithInsecure())
	require.Nil(t, err)
	defer func() {
		_ = cc.Close()
	}()

	client := registry.NewNetworkServiceRegistryClient(cc)

	stream, err := client.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "ns-1@" + remoteRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1@"+remoteRegistryDomain, list[0].Name)
}

/*
TestLocalDomain_NetworkServiceRegistry covers the next scenario:
	1. nsmgr from domain1 calls find with query "ns-1@domain1"
	2. local registry proxies query to proxy registry
	3. proxy registry proxies query to local registry removes interdomain symbol
	4. local registry finds ns-1 with local nsmgr URL

Expected: nsmgr found ns
domain1
 ____________________________________
|                                    |
| local registry <--> proxy registry |
|                                    |
_____________________________________
*/
func TestLocalDomain_NetworkServiceRegistry(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	const localRegistryDomain = "domain1.local.registry"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := new(sandbox.FakeDNSResolver)

	domain1 := sandbox.NewBuilder(t).
		SetContext(ctx).
		SetNodesCount(0).
		SetDNSDomainName(localRegistryDomain).
		SetDNSResolver(dnsServer).
		Build()
	defer domain1.Cleanup()

	require.NoError(t, dnsServer.Register(localRegistryDomain, domain1.Registry.URL))

	expected, err := domain1.Registry.NetworkServiceRegistryServer().Register(
		context.Background(),
		&registry.NetworkService{
			Name: "ns-1",
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithInsecure())
	require.Nil(t, err)
	defer func() {
		_ = cc.Close()
	}()
	client := registry.NewNetworkServiceRegistryClient(cc)

	stream, err := client.Find(context.Background(), &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: expected.Name + "@" + localRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, expected.Name, list[0].Name)
}

/*
	TestInterdomainFloatingNetworkServiceRegistry covers the next scenario:
		1. local registry from domain3 registers entry "ns-1"
		2. proxy registry from domain3 proxies entry "ns-1" to floating registry
		3. nsmgr from domain1 call find with query "ns-1@domain3"
		4. local registry from domain1 proxies query to proxy registry from domain1
		5. proxy registry from domain1 proxies query to floating registry
	Expected: nsmgr found ns
	domain1	                                        domain2                            domain3
	 ___________________________________            ___________________                ___________________________________
	|                                   | 2. Find  |                    | 1. Register |                                   |
	| local registry --> proxy registry | -------> | floating registry  | <---------  | proxy registry <-- local registry |
	|                                   |          |                    |             |                                   |
	____________________________________            ___________________                ___________________________________
*/

func TestInterdomainFloatingNetworkServiceRegistry(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.proxy.registry"
	const remoteRegistryDomain = "domain3.local.registry"
	const remoteProxyRegistryDomain = "domain3.proxy.registry"
	const floatingRegistryDomain = "domain2.floating.registry"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := new(sandbox.FakeDNSResolver)

	domain1 := sandbox.NewBuilder(t).
		SetContext(ctx).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()
	defer domain1.Cleanup()

	domain2 := sandbox.NewBuilder(t).
		SetContext(ctx).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()
	defer domain2.Cleanup()

	domain3 := sandbox.NewBuilder(t).
		SetNodesCount(0).
		SetRegistrySupplier(func(context.Context, *url.URL, ...grpc.DialOption) registry2.Registry {
			return registry2.NewServer(memory.NewNetworkServiceRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
		}).
		SetRegistryProxySupplier(nil).
		Build()
	defer domain3.Cleanup()

	require.NoError(t, dnsServer.Register(localRegistryDomain, domain1.Registry.URL))
	require.NoError(t, dnsServer.Register(proxyRegistryDomain, domain1.RegistryProxy.URL))
	require.NoError(t, dnsServer.Register(remoteRegistryDomain, domain2.Registry.URL))
	require.NoError(t, dnsServer.Register(remoteProxyRegistryDomain, domain2.RegistryProxy.URL))
	require.NoError(t, dnsServer.Register(floatingRegistryDomain, domain3.Registry.URL))

	_, err := domain2.Registry.NetworkServiceRegistryServer().Register(
		context.Background(),
		&registry.NetworkService{
			Name: "ns-1@" + floatingRegistryDomain,
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithInsecure())
	require.Nil(t, err)
	defer func() {
		_ = cc.Close()
	}()

	client := registry.NewNetworkServiceRegistryClient(cc)

	stream, err := client.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "ns-1@" + floatingRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "ns-1", list[0].Name)
}
