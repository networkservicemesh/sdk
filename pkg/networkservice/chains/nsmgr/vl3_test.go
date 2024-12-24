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

//go:build !windows
// +build !windows

package nsmgr_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/ipam/strictvl3ipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext/vl3dns"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	nscName = "nsc"
)

func Test_NSC_ConnectsTo_vl3NSE(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	dnsServerIPCh := make(chan net.IP, 1)
	dnsServerIPCh <- net.ParseIP("127.0.0.1")

	ipam := vl3.NewIPAM("10.0.0.1/24")

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ .NetworkService }}."),
			vl3dns.WithDNSPort(40053)),
		vl3.NewServer(ctx, ipam),
	)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:40053")
		},
	}

	for i := 0; i < 10; i++ {
		nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

		req := defaultRequest(nsReg.Name)
		req.Connection.Id = uuid.New().String()
		req.Connection.Labels["podName"] = nscName + fmt.Sprint(i)

		resp, err := nsc.Request(ctx, req)
		require.NoError(t, err)

		req.Connection = resp.Clone()
		require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), 1)
		require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)

		requireIPv4Lookup(ctx, t, &resolver, nscName+fmt.Sprint(i)+".vl3", "10.0.0.1")

		resp, err = nsc.Request(ctx, req)
		require.NoError(t, err)

		requireIPv4Lookup(ctx, t, &resolver, nscName+fmt.Sprint(i)+".vl3", "10.0.0.1")

		_, err = nsc.Close(ctx, resp)
		require.NoError(t, err)

		_, err = resolver.LookupIP(ctx, "ip4", nscName+fmt.Sprint(i)+".vl3")
		require.Error(t, err)
	}
}

func Test_vl3NSE_ConnectsTo_vl3NSE(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	var records genericsync.Map[string, []net.IP]
	var dnsServer = memory.NewDNSHandler(&records)

	records.Store("nsc1.vl3.", []net.IP{net.ParseIP("1.1.1.1")})

	dnsutils.ListenAndServe(ctx, dnsServer, ":40053")

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	var dnsConfigs = new(genericsync.Map[string, []*networkservice.DNSConfig])
	dnsServerIPCh := make(chan net.IP, 1)
	dnsServerIPCh <- net.ParseIP("0.0.0.0")

	serverIpam := vl3.NewIPAM("10.0.0.1/24")

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ .NetworkService }}."),
			vl3dns.WithDNSListenAndServeFunc(func(ctx context.Context, handler dnsutils.Handler, listenOn string) {
				dnsutils.ListenAndServe(ctx, handler, ":50053")
			}),
			vl3dns.WithConfigs(dnsConfigs),
			vl3dns.WithDNSPort(40053),
		),
		vl3.NewServer(ctx, serverIpam),
	)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:50053")
		},
	}

	clientIpam := vl3.NewIPAM("127.0.0.1/32")
	vl3ServerClient := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(vl3dns.NewClient(net.ParseIP("127.0.0.1"), dnsConfigs), vl3.NewClient(ctx, clientIpam)))

	req := defaultRequest(nsReg.Name)
	req.Connection.Id = uuid.New().String()

	req.Connection.Labels["podName"] = nscName

	resp, err := vl3ServerClient.Request(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)
	require.Equal(t, "127.0.0.1", resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps[0])

	require.Equal(t, "127.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	req.Connection = resp.Clone()

	requireIPv4Lookup(ctx, t, &resolver, "nsc.vl3", "127.0.0.1")

	requireIPv4Lookup(ctx, t, &resolver, "nsc1.vl3", "1.1.1.1") // we can lookup this ip address only and only if fanout is working

	resp, err = vl3ServerClient.Request(ctx, req)
	require.NoError(t, err)

	requireIPv4Lookup(ctx, t, &resolver, "nsc.vl3", "127.0.0.1")

	requireIPv4Lookup(ctx, t, &resolver, "nsc1.vl3", "1.1.1.1") // we can lookup this ip address only and only if fanout is working

	_, err = vl3ServerClient.Close(ctx, resp)
	require.NoError(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", "nsc.vl3")
	require.Error(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", "nsc1.vl3")
	require.Error(t, err)
}

func Test_NSC_GetsVl3DnsAddressDelay(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	dnsServerIPCh := make(chan net.IP, 1)

	ipam := vl3.NewIPAM("10.0.0.1/24")

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ .NetworkService }}."),
			vl3dns.WithDNSPort(40053)),
		vl3.NewServer(ctx, ipam))

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	req := defaultRequest(nsReg.Name)
	req.Connection.Labels["podName"] = nscName
	go func() {
		// Add a delay
		<-clock.FromContext(ctx).After(time.Millisecond * 200)
		dnsServerIPCh <- net.ParseIP("10.0.0.0")
	}()
	_, err = nsc.Request(ctx, req)
	require.NoError(t, err)
}

func Test_vl3NSE_ConnectsTo_Itself(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)
	dnsServerIPCh := make(chan net.IP, 1)

	ipam := vl3.NewIPAM("10.0.0.1/24")

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDNSPort(40053)),
		vl3.NewServer(ctx, ipam))

	// Connection to itself. This allows us to assign a dns address to ourselves.
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithName(nseReg.Name))
	req := defaultRequest(nsReg.Name)

	_, err = nsc.Request(ctx, req)
	require.NoError(t, err)
}

func Test_Interdomain_vl3_dns(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
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

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}

	dnsServerIPCh := make(chan net.IP, 1)
	dnsServerIPCh <- net.ParseIP("127.0.0.1")

	ipam := vl3.NewIPAM("10.0.0.1/24")

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken,
		vl3.NewServer(ctx, ipam),
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDNSPort(40053),
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ target .NetworkService }}.{{ domain .NetworkService }}."),
		),
		checkrequest.NewServer(t, func(t *testing.T, nsr *networkservice.NetworkServiceRequest) {
			require.False(t, interdomain.Is(nsr.GetConnection().GetNetworkService()))
		},
		),
	)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:40053")
		},
	}

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             uuid.New().String(),
			NetworkService: fmt.Sprint(nsReg.Name, "@", cluster2.Name),
			Labels:         map[string]string{"podName": nscName},
		},
	}

	resp, err := nsc.Request(ctx, req)
	require.NoError(t, err)

	req.Connection = resp.Clone()
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), 1)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].SearchDomains, 1)

	searchDomain := resp.GetContext().GetDnsContext().GetConfigs()[0].SearchDomains[0]
	requireIPv4Lookup(ctx, t, &resolver, fmt.Sprintf("%s.%s", nscName, searchDomain), "10.0.0.1")

	resp, err = nsc.Request(ctx, req)
	require.NoError(t, err)

	requireIPv4Lookup(ctx, t, &resolver, fmt.Sprintf("%s.%s", nscName, searchDomain), "10.0.0.1")

	_, err = nsc.Close(ctx, resp)
	require.NoError(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", fmt.Sprintf("%s.%s", nscName, searchDomain))
	require.Error(t, err)
}

func Test_FloatingInterdomain_vl3_dns(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
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

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3@"+floating.Name))
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint@" + floating.Name,
		NetworkServiceNames: []string{"vl3"},
	}

	dnsServerIPCh := make(chan net.IP, 1)
	dnsServerIPCh <- net.ParseIP("127.0.0.1")

	ipam := vl3.NewIPAM("10.0.0.1/24")

	cluster2.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken,
		vl3.NewServer(ctx, ipam),
		vl3dns.NewServer(ctx,
			dnsServerIPCh,
			vl3dns.WithDNSPort(40053),
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ target .NetworkService }}.{{ domain .NetworkService }}."),
		),
	)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:40053")
		},
	}

	nsc := cluster1.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             uuid.New().String(),
			NetworkService: fmt.Sprint(nsReg.Name),
			Labels:         map[string]string{"podName": nscName},
		},
	}

	resp, err := nsc.Request(ctx, req)
	require.NoError(t, err)

	req.Connection = resp.Clone()
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), 1)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].SearchDomains, 3)

	searchDomain := resp.GetContext().GetDnsContext().GetConfigs()[0].SearchDomains[0]

	requireIPv4Lookup(ctx, t, &resolver, fmt.Sprintf("%s.%s", nscName, searchDomain), "10.0.0.1")

	resp, err = nsc.Request(ctx, req)
	require.NoError(t, err)

	requireIPv4Lookup(ctx, t, &resolver, fmt.Sprintf("%s.%s", nscName, searchDomain), "10.0.0.1")

	_, err = nsc.Close(ctx, resp)
	require.NoError(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", fmt.Sprintf("%s.%s", nscName, searchDomain))
	require.Error(t, err)
}

func Test_NSC_ConnectsTo_vl3NSE_With_Invalid_IpContext(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		SetNSMgrProxySupplier(nil).
		SetRegistryProxySupplier(nil).
		Build()

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	prefix1 := "10.0.0.0/24"
	prefix2 := "10.10.0.0/24"

	serverIpam := vl3.NewIPAM(prefix1)

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		strictvl3ipam.NewServer(ctx, vl3.NewServer, serverIpam),
	)

	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)

	req := defaultRequest(nsReg.Name)
	conn, err := nsc.Request(ctx, req)
	require.NoError(t, err)
	require.True(t, checkIPContext(conn.Context.IpContext, prefix1))

	// Refresh
	clonedConn := conn.Clone()
	req.Connection = conn
	conn, err = nsc.Request(ctx, req)
	require.NoError(t, err)
	require.True(t, checkIPContext(conn.Context.IpContext, prefix1))
	require.True(t, proto.Equal(clonedConn.GetContext().IpContext, conn.GetContext().IpContext))

	// Reset ipam with a new prefix
	err = serverIpam.Reset(prefix2)
	require.NoError(t, err)

	req.Connection = conn
	conn, err = nsc.Request(ctx, req)
	require.NoError(t, err)

	require.False(t, checkIPContext(conn.Context.IpContext, prefix1))
	require.True(t, checkIPContext(conn.Context.IpContext, prefix2))

	// Refresh
	clonedConn = conn.Clone()
	req.Connection = conn
	conn, err = nsc.Request(ctx, req)
	require.NoError(t, err)
	require.True(t, checkIPContext(conn.Context.IpContext, prefix2))
	require.True(t, proto.Equal(clonedConn.GetContext().IpContext, conn.GetContext().IpContext))
}

func checkIPContext(ipContext *networkservice.IPContext, prefix string) bool {
	pool := ippool.NewWithNetString(prefix)
	for _, addr := range ipContext.SrcIpAddrs {
		if !pool.ContainsNetString(addr) {
			return false
		}
	}
	for _, addr := range ipContext.DstIpAddrs {
		if !pool.ContainsNetString(addr) {
			return false
		}
	}
	return true
}
