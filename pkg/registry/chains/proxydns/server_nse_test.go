// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	registryapi "github.com/networkservicemesh/api/pkg/api/registry"

	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

/*
TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
 1. local registry from domain2 has entry "nse-1"
 2. nsmgr from domain1 call find with query "nse-1@domain2"
 3. local registry proxies query to nsmgr proxy registry
 4. nsmgr proxy registry proxies query to proxy registry
 5. proxy registry proxies query to local registry from domain2

Expected: nsmgr found nse

domain1:                                                           domain2:
------------------------------------------------------------       -------------------
|                                                           | Find |                 |
|  local registry -> nsmgr proxy registry -> proxy registry | ---> | local registry  |
|                                                           |      |                 |
------------------------------------------------------------       ------------------.
*/
func TestInterdomainNetworkServiceEndpointRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	dnsServer := sandbox.NewFakeResolver()

	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		Build()

	domain2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("domain2").
		Build()

	expirationTime := timestamppb.New(time.Now().Add(time.Hour))

	registryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain2.Registry.URL))

	reg, err := registryClient.Register(
		context.Background(),
		&registryapi.NetworkServiceEndpoint{
			Name:           "nse-1",
			Url:            "nsmgr-url",
			ExpirationTime: expirationTime,
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)

	defer func() {
		_ = cc.Close()
	}()

	client := registryapi.NewNetworkServiceEndpointRegistryClient(cc)

	stream, err := client.Find(ctx, &registryapi.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registryapi.NetworkServiceEndpoint{
			Name: reg.GetName() + "@" + domain2.Name,
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceEndpointList(stream)

	require.Len(t, list, 1)
	require.Equal(t, reg.GetName()+"@"+domain2.Name, list[0].GetName())
}

/*
TestLocalDomain_NetworkServiceEndpointRegistry covers the next scenario:
 1. nsmgr from domain1 calls find with query "nse-1@domain1"
 2. local registry proxies query to nsmgr proxy registry
 3. nsmgr proxy registry  proxies query to proxy registry
 4. proxy registry proxies query to local registry

Expected: nsmgr found nse

domain1:
-----------------------------------------------------------------------------------
|                                                                                 |
|    local registry -> nsmgr proxy registry -> proxy registry -> local registry   |
|                                                                                 |
-----------------------------------------------------------------------------------.
*/
func TestLocalDomain_NetworkServiceEndpointRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	dnsServer := sandbox.NewFakeResolver()

	domain1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("cluster.local").
		SetDNSResolver(dnsServer).
		Build()

	expirationTime := timestamppb.New(time.Now().Add(time.Hour))

	registryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain1.Registry.URL))

	reg, err := registryClient.Register(
		context.Background(),
		&registryapi.NetworkServiceEndpoint{
			Name:           "nse-1",
			Url:            "test://publicNSMGRurl",
			ExpirationTime: expirationTime,
		},
	)
	require.Nil(t, err)

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)

	defer func() {
		_ = cc.Close()
	}()

	client := registryapi.NewNetworkServiceEndpointRegistryClient(cc)

	stream, err := client.Find(context.Background(), &registryapi.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registryapi.NetworkServiceEndpoint{
			Name: reg.GetName() + "@" + domain1.Name,
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceEndpointList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "nse-1@"+domain1.Name, list[0].GetName())
}

/*
TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
 1. local registry from domain2 has entry "nse-1"
 2. nsmgr from domain1 call find with query "nse-1@domain2"
 3. local registry proxies query to nsmgr proxy registry
 4. nsmgr proxy registry proxies query to proxy registry
 5. proxy registry proxies query to local registry from domain2

Expected: nsmgr found nse

domain1:                                                             domain2:
------------------------------------------------------------         -------------------              ------------------------------------------------------------
|                                                           | 2.Find |                 | 1. Register  |                                                           |
|  local registry -> nsmgr proxy registry -> proxy registry |  --->  | local registry  | <----------- |  local registry -> nsmgr proxy registry -> proxy registry |
|                                                           |        |                 |              |                                                           |
------------------------------------------------------------         ------------------               ------------------------------------------------------------.
*/
func TestInterdomainFloatingNetworkServiceEndpointRegistry(t *testing.T) {
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
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		SetDNSDomainName("floating.domain").
		Build()

	expirationTime := timestamppb.New(time.Now().Add(time.Hour))

	registryClient := registryclient.NewNetworkServiceEndpointRegistryClient(ctx,
		registryclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
		registryclient.WithClientURL(domain2.Registry.URL))

	reg, err := registryClient.Register(
		context.Background(),
		&registryapi.NetworkServiceEndpoint{
			Name:           "nse-1@" + domain3.Name,
			Url:            "test://publicNSMGRurl",
			ExpirationTime: expirationTime,
		},
	)
	require.Nil(t, err)

	name := strings.Split(reg.GetName(), "@")[0]

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(domain1.Registry.URL), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)

	defer func() {
		_ = cc.Close()
	}()

	client := registryapi.NewNetworkServiceEndpointRegistryClient(cc)

	stream, err := client.Find(ctx, &registryapi.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registryapi.NetworkServiceEndpoint{
			Name: name + "@" + domain3.Name,
		},
	})

	require.Nil(t, err)

	list := registryapi.ReadNetworkServiceEndpointList(stream)

	require.Len(t, list, 1)
	require.Equal(t, name+"@"+domain3.Name, list[0].GetName())
}
