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

package nsmgrproxy_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgrproxy"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	kernelmech "github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	registryclient "github.com/networkservicemesh/sdk/pkg/registry/chains/client"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// Test_NsmgrProxy_ShouldAbleToUseLoadBalancerURls covers a scenario when we need to change nse.URL of registration to something else.
// For example to load balancer address.
func Test_NsmgrProxy_ShouldAbleToUseLoadBalancerURls(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = new(sandbox.FakeDNSResolver)

	var supplyNSMGRProxy sandbox.SupplyNSMgrProxyFunc = func(ctx context.Context, regURL, proxyURL *url.URL, tokenGenerator token.GeneratorFunc, options ...nsmgrproxy.Option) nsmgr.Nsmgr {
		var p = filepath.Join(t.TempDir(), "urls-map.yaml")
		require.NoError(t, ioutil.WriteFile(p, []byte(fmt.Sprintf("%v: %v", regURL.Hostname(), "my-lb-dns-name")), os.ModePerm))
		return nsmgrproxy.NewServer(ctx, regURL, proxyURL, tokenGenerator, append(options, nsmgrproxy.WithMapURLFilePath(p))...)
	}

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		SetNSMgrProxySupplier(supplyNSMGRProxy).
		Build()

	floatingDomain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("floating.domain").
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floatingDomain.Name,
		NetworkServiceNames: []string{"my-service-interdomain"},
	}

	cluster1.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	var list []*registry.NetworkServiceEndpoint
	var floatingRegistryClient = registryclient.NewNetworkServiceEndpointRegistryClient(ctx, registryclient.WithDialOptions(sandbox.DialOptions()...), registryclient.WithClientURL(floatingDomain.Registry.URL))

	require.Eventually(t, func() bool {
		var stream, err = floatingRegistryClient.Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "final-endpoint"}})
		if err != nil {
			return false
		}
		list = registry.ReadNetworkServiceEndpointList(stream)
		return len(list) > 0
	}, time.Second, time.Second/20)

	require.Equal(t, "tcp://my-lb-dns-name:"+cluster1.NSMgrProxy.URL.Port(), list[0].Url)
}

// TestNSMGR_InterdomainUseCase covers simple interdomain scenario:
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse3
func TestNSMGR_InterdomainUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain",
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken, checkrequest.NewServer(t, func(t *testing.T, nsr *networkservice.NetworkServiceRequest) {
		require.False(t, interdomain.Is(nsr.GetConnection().GetNetworkService()))
	}))

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name, "@", cluster2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_InterdomainUseCase covers simple interdomain scenario:
//
// request1: nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> final-endpoint via cluster2
// request2: nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> final-endpoint via floating registry
func Test_NSEMovedFromInterdomainToFloatingUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	floating := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("floating.domain").
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()
	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg1 := &registry.NetworkService{
		Name: "my-service-interdomain",
	}

	var err error

	nsReg1, err = nsRegistryClient.Register(ctx, nsReg1)
	require.NoError(t, err)

	nsReg2 := &registry.NetworkService{
		Name: "my-service-interdomain@" + floating.Name,
	}

	nsReg2, err = nsRegistryClient.Register(ctx, nsReg2)
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg1.Name},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{nsReg1.Name},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	stream, err := adapters.NetworkServiceEndpointServerToClient(cluster2.Registry.NetworkServiceEndpointRegistryServer()).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: nseReg1.Name,
	}})
	require.NoError(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 1)

	stream, err = adapters.NetworkServiceEndpointServerToClient(cluster1.Registry.NetworkServiceEndpointRegistryServer()).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: nseReg1.Name,
	}})
	require.NoError(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 0)

	stream, err = adapters.NetworkServiceEndpointServerToClient(floating.Registry.NetworkServiceEndpointRegistryServer()).Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: nseReg1.Name,
	}})
	require.NoError(t, err)
	require.Len(t, registry.ReadNetworkServiceEndpointList(stream), 1)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	var finalNSE = map[string]string{
		fmt.Sprint(nsReg1.Name, "@", cluster2.Name): nseReg1.GetName(),
		nsReg2.GetName(): nseReg2.GetName(),
	}

	for _, nsName := range []string{fmt.Sprint(nsReg1.Name, "@", cluster2.Name), nsReg2.GetName()} {
		request := &networkservice.NetworkServiceRequest{
			MechanismPreferences: []*networkservice.Mechanism{
				{Cls: cls.LOCAL, Type: kernel.MECHANISM},
			},
			Connection: &networkservice.Connection{
				Id:             "1",
				NetworkService: nsName,
				Context:        &networkservice.ConnectionContext{},
			},
		}

		conn, err := nsc.Request(ctx, request)
		require.NoError(t, err)
		require.NotNil(t, conn)

		require.Equal(t, 8, len(conn.Path.PathSegments))

		require.Equal(t, finalNSE[nsName], conn.GetPath().GetPathSegments()[7].GetName())

		// Simulate refresh from client.

		refreshRequest := request.Clone()
		refreshRequest.Connection = conn.Clone()

		conn, err = nsc.Request(ctx, refreshRequest)
		require.NoError(t, err)
		require.NotNil(t, conn)
		require.Equal(t, 8, len(conn.Path.PathSegments))
		require.Equal(t, finalNSE[nsName], conn.GetPath().GetPathSegments()[7].GetName())

		// Close
		_, err = nsc.Close(ctx, conn)
		require.NoError(t, err)
	}
}

// TestNSMGR_Interdomain_TwoNodesNSEs covers scenarion with connection from the one client to two endpoints from diffrenret clusters.
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse2
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3
func TestNSMGR_Interdomain_TwoNodesNSEs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(2).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-service-interdomain-1",
	})
	require.NoError(t, err)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-service-interdomain-2",
	})
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-1",
		NetworkServiceNames: []string{"my-service-interdomain-1"},
	}
	cluster2.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-2",
		NetworkServiceNames: []string{"my-service-interdomain-2"},
	}
	cluster2.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint("my-service-interdomain-1", "@", cluster2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	request = &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "2",
			NetworkService: fmt.Sprint("my-service-interdomain-2", "@", cluster2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest = request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))
}

// TestNSMGR_FloatingInterdomainUseCase covers simple interdomain scenario with resolving endpoint from floating registry:
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse3
func TestNSMGR_FloatingInterdomainUseCase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	floating := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("floating.domain").
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain"},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	c := adapters.NetworkServiceEndpointServerToClient(cluster2.Nodes[0].NSMgr.NetworkServiceEndpointRegistryServer())

	s, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: "final-endpoint@" + floating.Name,
	}})

	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(s)

	require.Len(t, list, 1)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_FloatingInterdomainUseCase_FloatingNetworkServiceNameRegistration covers simple interdomain scenario with resolving endpoint from floating registry:
//
// registration: {"name": "nse@floating.domain", "networkservice_names": ["my-service-interdomain@floating.domain"]}
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse3
func TestNSMGR_FloatingInterdomainUseCase_FloatingNetworkServiceNameRegistration(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	floating := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("floating.domain").
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := &registry.NetworkService{
		Name: "my-service-interdomain@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain@" + floating.Name},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}

// TestNSMGR_FloatingInterdomain_FourClusters covers scenarion with connection from the one client to two endpoints
// from diffrenret clusters using floating registry for resolving endpoints.
//
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy2 -> nsmgr2 ->forwarder2 -> nsmgr2 -> nse2
//	nsc -> nsmgr1 ->  forwarder1 -> nsmgr1 -> nsmgr-proxy1 -> nsmg-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3
func TestNSMGR_FloatingInterdomain_FourClusters(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	var dnsServer = sandbox.NewFakeResolver()

	// setup clusters

	cluster1 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSResolver(dnsServer).
		SetDNSDomainName("cluster1").
		Build()

	cluster2 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster2").
		SetDNSResolver(dnsServer).
		Build()

	cluster3 := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetDNSDomainName("cluster3").
		SetDNSResolver(dnsServer).
		Build()

	floating := sandbox.NewBuilder(ctx, t).
		SetNodesCount(0).
		SetDNSDomainName("floating.domain").
		SetDNSResolver(dnsServer).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	// register first ednpoint

	nsRegistryClient := cluster2.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg1 := &registry.NetworkService{
		Name: "my-service-interdomain-1@" + floating.Name,
	}

	_, err := nsRegistryClient.Register(ctx, nsReg1)
	require.NoError(t, err)

	nseReg1 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-1@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain-1"},
	}

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg1, sandbox.GenerateTestToken)

	nsReg2 := &registry.NetworkService{
		Name: "my-service-interdomain-1@" + floating.Name,
	}

	// register second ednpoint

	nsRegistryClient = cluster3.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err = nsRegistryClient.Register(ctx, nsReg2)
	require.NoError(t, err)

	nseReg2 := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint-2@" + floating.Name,
		NetworkServiceNames: []string{"my-service-interdomain-2"},
	}

	cluster3.Nodes[0].NewEndpoint(ctx, nseReg2, sandbox.GenerateTestToken)

	// connect to first endpoint from cluster2

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprint(nsReg1.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))

	// connect to second endpoint from cluster3
	request = &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "2",
			NetworkService: fmt.Sprint(nsReg2.Name),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err = nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest = request.Clone()
	refreshRequest.Connection = conn.Clone()

	conn, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 8, len(conn.Path.PathSegments))
}

type passThroughClient struct {
	networkService string
}

func newPassTroughClient(networkService string) *passThroughClient {
	return &passThroughClient{
		networkService: networkService,
	}
}

func (c *passThroughClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	request.Connection.NetworkService = c.networkService
	request.Connection.NetworkServiceEndpointName = ""
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *passThroughClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	conn.NetworkService = c.networkService
	return next.Client(ctx).Close(ctx, conn, opts...)
}

// Test_Interdomain_PassThroughUsecase covers scenario when we have 5 clusters.
// Each cluster contains NSE with name endpoint-${cluster-num}.
// Each endpoint request endpoint from the previous cluster (exclude 1st).
//
//
// nsc -> nsmgr4 ->  forwarder4 -> nsmgr4 -> nsmgr-proxy4 -> nsmgr-proxy3 -> nsmgr3 ->forwarder3 -> nsmgr3 -> nse3 ->
// nse3 -> nsmgr3 ->  forwarder3 -> nsmgr3 -> nsmgr-proxy3 -> nsmgr-proxy2 -> nsmgr2 ->forwarder2-> nsmgr2 -> nse2 ->
// nse2 -> nsmgr2 ->  forwarder2 -> nsmgr2 -> nsmgr-proxy2 -> nsmgr-proxy1 -> nsmgr1 -> forwarder1 -> nsmgr1 -> nse1 ->
// nse1 -> nsmgr1 ->  forwarder1 -> nsmg1 -> nsmgr-proxy1 -> nsmgr-proxy0 -> nsmgr0 -> forwarder0 -> nsmgr0 -> nse0

func Test_Interdomain_PassThroughUsecase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const clusterCount = 5

	var dnsServer = sandbox.NewFakeResolver()
	var clusters = make([]*sandbox.Domain, clusterCount)

	for i := 0; i < clusterCount; i++ {
		clusters[i] = sandbox.NewBuilder(ctx, t).
			SetNodesCount(1).
			SetDNSResolver(dnsServer).
			SetDNSDomainName("cluster" + fmt.Sprint(i)).
			Build()
		var additionalFunctionality []networkservice.NetworkServiceServer
		if i != 0 {
			// Passtrough to the node i-1
			additionalFunctionality = []networkservice.NetworkServiceServer{
				chain.NewNetworkServiceServer(
					clienturl.NewServer(clusters[i].Nodes[0].NSMgr.URL),
					connect.NewServer(
						client.NewClient(
							ctx,
							client.WithAdditionalFunctionality(
								newPassTroughClient(fmt.Sprintf("my-service-remote-%v@cluster%v", i-1, i-1)),
								kernelmech.NewClient(),
							),
							client.WithDialTimeout(sandbox.DialTimeout),
							client.WithDialOptions(sandbox.DialOptions()...),
							client.WithoutRefresh(),
						),
					),
				),
			}
		}

		nsRegistryClient := clusters[i].NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

		nsReg, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
			Name: fmt.Sprintf("my-service-remote-%v", i),
		})
		require.NoError(t, err)

		nsesReg := &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("endpoint-%v", i),
			NetworkServiceNames: []string{nsReg.Name},
		}
		clusters[i].Nodes[0].NewEndpoint(ctx, nsesReg, sandbox.GenerateTestToken, additionalFunctionality...)
	}

	nsc := clusters[clusterCount-1].Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: fmt.Sprintf("my-service-remote-%v", clusterCount-1),
			Context:        &networkservice.ConnectionContext{},
		},
	}

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Path length to first endpoint is 4
	// Path length from NSE client to other remote endpoint is 8
	require.Equal(t, 8*(clusterCount-1)+4, len(conn.Path.PathSegments))

	// Close
	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)
}
