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

package nsmgr_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/nsmgr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/tools/clientinfo"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/cache"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/norecursion"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func requireIPv4Lookup(ctx context.Context, t *testing.T, r *net.Resolver, host, expected string) {
	addrs, err := r.LookupIP(ctx, "ip4", host)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	require.Equal(t, expected, addrs[0].String())
}

func Test_DNSUsecase(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*200)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService(t.Name()))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	dnsConfigsMap := new(dnsconfigs.Map)
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(dnscontext.NewClient(
		dnscontext.WithChainContext(ctx),
		dnscontext.WithDNSConfigsMap(dnsConfigsMap),
	)))

	dnsConfigs := []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{"127.0.0.1:40053"},
			SearchDomains: []string{"com"},
		},
	}

	// DNS server on nse side
	dnsRecords := new(memory.Map)
	dnsRecords.Store("my.domain.", []net.IP{net.ParseIP("4.4.4.4")})
	dnsRecords.Store("my.domain.com.", []net.IP{net.ParseIP("5.5.5.5")})
	dnsutils.ListenAndServe(ctx, memory.NewDNSHandler(dnsRecords), ":40053")

	// DNS server on nsc side
	clientDNSHandler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(dnsConfigsMap),
		searches.NewDNSHandler(),
		noloop.NewDNSHandler(),
		norecursion.NewDNSHandler(),
		cache.NewDNSHandler(),
		fanout.NewDNSHandler(fanout.WithTimeout(time.Second)),
	)
	dnsutils.ListenAndServe(ctx, clientDNSHandler, ":50053")

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: nsReg.Name,
			Context: &networkservice.ConnectionContext{
				DnsContext: &networkservice.DNSContext{
					Configs: dnsConfigs,
				},
			},
			Labels: make(map[string]string),
		},
	}

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:50053")
		},
	}

	_, err = resolver.LookupIP(ctx, "ip4", "my.domain")
	require.Error(t, err)

	conn, err := nsc.Request(ctx, request)
	require.NoError(t, err)

	requireIPv4Lookup(ctx, t, &resolver, "my.domain", "4.4.4.4")
	requireIPv4Lookup(ctx, t, &resolver, "my.domain.com", "5.5.5.5")

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func Test_AwareNSEs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	const count = 3
	var nseRegs [count]*registry.NetworkServiceEndpoint
	var nses [count]*sandbox.EndpointEntry
	var requests [count]*networkservice.NetworkServiceRequest

	ns1 := defaultRegistryService("my-ns-1")
	ns2 := defaultRegistryService("my-ns-2")

	nsurl1, err := url.Parse(fmt.Sprintf("kernel://%s?%s=%s", ns1.Name, "color", "red"))
	require.NoError(t, err)

	nsurl2, err := url.Parse(fmt.Sprintf("kernel://%s?%s=%s", ns2.Name, "color", "red"))
	require.NoError(t, err)

	nsInfo := [count]struct {
		ns         *registry.NetworkService
		labelKey   string
		labelValue string
	}{
		{
			ns:         ns1,
			labelKey:   "color",
			labelValue: "red",
		},
		{
			ns:         ns2,
			labelKey:   "color",
			labelValue: "red",
		},
		{
			ns:         ns1,
			labelKey:   "day",
			labelValue: "friday",
		},
	}

	for i := 0; i < count; i++ {
		nseRegs[i] = &registry.NetworkServiceEndpoint{
			Name:                fmt.Sprintf("nse-%s", uuid.New().String()),
			NetworkServiceNames: []string{nsInfo[i].ns.Name},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				nsInfo[i].ns.Name: {
					Labels: map[string]string{
						nsInfo[i].labelKey: nsInfo[i].labelValue,
					},
				},
			},
		}

		nses[i] = domain.Nodes[0].NewEndpoint(ctx, nseRegs[i], sandbox.GenerateTestToken, point2pointipam.NewServer(ipNet))

		requests[i] = &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             fmt.Sprint(i),
				NetworkService: nsInfo[i].ns.Name,
				Context:        &networkservice.ConnectionContext{},
				Mechanism:      &networkservice.Mechanism{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
				Labels: map[string]string{
					nsInfo[i].labelKey: nsInfo[i].labelValue,
				},
			},
		}

		nsInfo[i].ns.Matches = append(nsInfo[i].ns.Matches,
			&registry.Match{
				SourceSelector: map[string]string{nsInfo[i].labelKey: nsInfo[i].labelValue},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{nsInfo[i].labelKey: nsInfo[i].labelValue},
					},
				},
			},
		)
	}

	_, err = nsRegistryClient.Register(ctx, ns1)
	require.NoError(t, err)
	_, err = nsRegistryClient.Register(ctx, ns2)
	require.NoError(t, err)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(
		excludedprefixes.NewClient(excludedprefixes.WithAwarenessGroups(
			[][]*url.URL{
				{nsurl1, nsurl2},
			},
		))))

	var conns [count]*networkservice.Connection
	for i := 0; i < count; i++ {
		conns[i], err = nsc.Request(ctx, requests[i])
		require.NoError(t, err)
		require.Equal(t, conns[0].NetworkServiceEndpointName, nses[0].Name)
	}

	srcIP1 := conns[0].GetContext().GetIpContext().GetSrcIpAddrs()
	srcIP2 := conns[1].GetContext().GetIpContext().GetSrcIpAddrs()
	srcIP3 := conns[2].GetContext().GetIpContext().GetSrcIpAddrs()

	require.Equal(t, srcIP1[0], srcIP2[0])
	require.NotEqual(t, srcIP1[0], srcIP3[0])
	require.NotEqual(t, srcIP2[0], srcIP3[0])

	for i := 0; i < count; i++ {
		_, err = nsc.Close(ctx, conns[i])
		require.NoError(t, err)
	}

	for i := 0; i < count; i++ {
		_, err = nses[i].Unregister(ctx, nseRegs[i])
		require.NoError(t, err)
	}
}

func Test_ShouldParseNetworkServiceLabelsTemplate(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const (
		testEnvName             = "NODE_NAME"
		testEnvValue            = "testValue"
		destinationTestKey      = `nodeName`
		destinationTestTemplate = `{{.nodeName}}`
	)

	err := os.Setenv(testEnvName, testEnvValue)
	require.NoError(t, err)

	want := map[string]string{}
	clientinfo.AddClientInfo(ctx, want)

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNSMgrProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg := defaultRegistryService(t.Name())
	nsReg.Matches = []*registry.Match{
		{
			Routes: []*registry.Destination{
				{
					DestinationSelector: map[string]string{
						destinationTestKey: destinationTestTemplate,
					},
				},
			},
		},
	}

	nsReg, err = nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	nseReg.NetworkServiceLabels = map[string]*registry.NetworkServiceLabels{nsReg.Name: {}}

	nse := domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	require.NoError(t, err)

	req := defaultRequest(nsReg.Name)

	conn, err := nsc.Request(ctx, req)
	require.NoError(t, err)

	// Test for connection labels setting
	require.Equal(t, want, conn.Labels)
	// Test for endpoint labels setting
	require.Equal(t, want, nseReg.NetworkServiceLabels[nsReg.Name].Labels)

	_, err = nse.Unregister(ctx, nseReg)
	require.NoError(t, err)
}

func Test_UsecasePoint2MultiPoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		SetRegistryExpiryDuration(time.Second).
		Build()

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "p2mp forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"p2mp": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "p2p forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"p2p": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                "special forwarder",
		NetworkServiceNames: []string{"forwarder"},
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"forwarder": {
				Labels: map[string]string{
					"special": "true",
				},
			},
		},
	}, sandbox.GenerateTestToken)

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{},
					},
				},
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, "p2mp forwarder", conn.GetPath().GetPathSegments()[2].Name)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{},
					},
				},
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						// no labels
					},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 4, len(conn.Path.PathSegments))
	require.Equal(t, "p2p forwarder", conn.GetPath().GetPathSegments()[2].Name)
}
func Test_RemoteUsecase_Point2MultiPoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	const nodeCount = 2

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(nodeCount).
		SetRegistryProxySupplier(nil).
		SetNodeSetup(func(ctx context.Context, node *sandbox.Node, _ int) {
			node.NewNSMgr(ctx, "nsmgr", nil, sandbox.GenerateTestToken, nsmgr.NewServer)
		}).
		SetRegistryExpiryDuration(time.Second).
		Build()

	for i := 0; i < nodeCount; i++ {
		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "p2mp forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "p2p forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"p2p": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)

		domain.Nodes[i].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
			Name:                "special forwarder-" + fmt.Sprint(i),
			NetworkServiceNames: []string{"forwarder"},
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"forwarder": {
					Labels: map[string]string{
						"special": "true",
					},
				},
			},
		}, sandbox.GenerateTestToken)
	}
	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	_, err := nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						"p2mp": "true",
					},
				},
			},
		},
	})
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "my-nse-1",
		NetworkServiceNames: []string{"my-ns"},
	}

	domain.Nodes[1].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	request := defaultRequest("my-ns")

	conn, err := nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.Path.PathSegments))
	require.Equal(t, "p2mp forwarder-0", conn.GetPath().GetPathSegments()[2].Name)
	require.Equal(t, "p2mp forwarder-1", conn.GetPath().GetPathSegments()[4].Name)

	_, err = nsRegistryClient.Register(ctx, &registry.NetworkService{
		Name: "my-ns",
		Matches: []*registry.Match{
			{
				Metadata: &registry.Metadata{
					Labels: map[string]string{
						// no labels
					},
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = nsc.Close(ctx, conn)
	require.NoError(t, err)

	conn, err = nsc.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.NotNil(t, conn)
	require.Equal(t, 6, len(conn.Path.PathSegments))
	require.Equal(t, "p2p forwarder-0", conn.GetPath().GetPathSegments()[2].Name)
	require.Equal(t, "p2p forwarder-1", conn.GetPath().GetPathSegments()[4].Name)
}
