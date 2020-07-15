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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dnsresolve"
	"github.com/networkservicemesh/sdk/pkg/registry/common/proxy"
	"github.com/networkservicemesh/sdk/pkg/registry/common/swap"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
)

func localNSERegistryServer(proxyRegistryURL *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return next.NewNetworkServiceEndpointRegistryServer(
		memory.NewNetworkServiceEndpointRegistryServer(),
		proxy.NewNetworkServiceEndpointRegistryServer(proxyRegistryURL),
		connect.NewNetworkServiceEndpointRegistryServer(func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient {
			return registry.NewNetworkServiceEndpointRegistryClient(cc)
		}, connect.WithExpirationDuration(time.Millisecond*500), connect.WithClientDialOptions(grpc.WithInsecure())),
	)
}

func proxyNSERegistryServer(currentDomain string, resolver dnsresolve.Resolver) registry.NetworkServiceEndpointRegistryServer {
	return next.NewNetworkServiceEndpointRegistryServer(
		dnsresolve.NewNetworkServiceEndpointRegistryServer(dnsresolve.WithResolver(resolver)),
		swap.NewNetworkServiceEndpointRegistryServer(currentDomain, &url.URL{Scheme: "test", Host: "proxyNSMGRurl"}, &url.URL{Scheme: "test", Host: "publicNSMGRurl"}),
		connect.NewNetworkServiceEndpointRegistryServer(func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient {
			return registry.NewNetworkServiceEndpointRegistryClient(cc)
		}, connect.WithExpirationDuration(time.Millisecond*500), connect.WithClientDialOptions(grpc.WithInsecure())),
	)
}

/*
	TestInterdomainNetworkServiceEndpointRegistry covers the next scenario:
		1. local registry from domain2 has entry "ns-1"
		2. nsmgr from domain1 call find with query "nse-1@domain2"
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
func TestInterdomainNetworkServiceEndpointRegistry(t *testing.T) {
	tool := newInterdomainTestingTool(t)
	defer tool.verifyNoneLeaks()
	defer tool.cleanup()

	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.proxy.registry"
	const remoteRegistryDomain = "domain2.local.registry"

	proxyRegistryURL := tool.startNetworkServiceEndpointRegistryServerAsync(proxyRegistryDomain, proxyNSERegistryServer(localRegistryDomain, tool))
	tool.startNetworkServiceEndpointRegistryServerAsync(localRegistryDomain, localNSERegistryServer(proxyRegistryURL))

	remoteMem := memory.NewNetworkServiceEndpointRegistryServer()
	_, err := remoteMem.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1", Url: "nsmgr-url"})
	require.Nil(t, err)

	tool.startNetworkServiceEndpointRegistryServerAsync(remoteRegistryDomain, remoteMem)

	client := registry.NewNetworkServiceEndpointRegistryClient(tool.dialDomain(localRegistryDomain))

	stream, err := client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse-1@" + remoteRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "nse-1@nsmgr-url", list[0].Name)
}

/*
	TestInterdomainFloatingNetworkServiceEndpointRegistry covers the next scenario:
		1. local registry from domain3 registers entry "ns-1"
		2. proxy registry from domain3 proxies entry "ns-1" to floating registry
		3. nsmgr from domain1 call find with query "nse-1@domain3"
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

func TestInterdomainFloatingNetworkServiceEndpointRegistry(t *testing.T) {
	tool := newInterdomainTestingTool(t)
	defer tool.verifyNoneLeaks()
	defer tool.cleanup()
	const localRegistryDomain = "domain1.local.registry"
	const proxyRegistryDomain = "domain1.proxy.registry"
	const remoteRegistryDomain = "domain3.local.registry"
	const remoteProxyRegistryDomain = "domain3.proxy.registry"
	const floatingRegistryDomain = "domain2.floating.registry"

	fMem := memory.NewNetworkServiceEndpointRegistryServer()

	proxyRegistryURL1 := tool.startNetworkServiceEndpointRegistryServerAsync(proxyRegistryDomain, proxyNSERegistryServer(localRegistryDomain, tool))
	tool.startNetworkServiceEndpointRegistryServerAsync(localRegistryDomain, localNSERegistryServer(proxyRegistryURL1))

	proxyRegistryURL2 := tool.startNetworkServiceEndpointRegistryServerAsync(remoteProxyRegistryDomain, proxyNSERegistryServer(remoteRegistryDomain, tool))
	tool.startNetworkServiceEndpointRegistryServerAsync(remoteRegistryDomain, localNSERegistryServer(proxyRegistryURL2))

	tool.startNetworkServiceEndpointRegistryServerAsync(floatingRegistryDomain, fMem)

	domain2Client := registry.NewNetworkServiceEndpointRegistryClient(tool.dialDomain(remoteRegistryDomain))
	_, err := domain2Client.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1@" + floatingRegistryDomain,
	})
	require.Nil(t, err)

	fStream, err := adapters.NetworkServiceEndpointServerToClient(fMem).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{}})
	require.Nil(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(fStream), 1)

	domain1Client := registry.NewNetworkServiceEndpointRegistryClient(tool.dialDomain(localRegistryDomain))

	stream, err := domain1Client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse-1@" + floatingRegistryDomain,
		},
	})

	require.Nil(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)

	require.Len(t, list, 1)
	require.Equal(t, "nse-1@test://publicNSMGRurl", list[0].Name)
}
