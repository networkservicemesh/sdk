// Copyright (c) 2022 Cisco and/or its affiliates.
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

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext/vl3dns"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
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

	var serverPrefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(serverPrefixCh)

	serverPrefixCh <- &ipam.PrefixResponse{Prefix: "10.0.0.1/24"}

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3.NewServer(ctx, serverPrefixCh),
		vl3dns.NewServer(ctx,
			func() net.IP { return net.ParseIP("127.0.0.1") },
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ .NetworkService }}."),
			vl3dns.WithDNSPort(40053)),
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

		reqCtx, reqClose := context.WithTimeout(ctx, time.Second)
		defer reqClose()

		req := defaultRequest(nsReg.Name)
		req.Connection.Id = uuid.New().String()

		req.Connection.Labels["podName"] = "nsc" + fmt.Sprint(i)

		resp, err := nsc.Request(reqCtx, req)

		require.NoError(t, err)
		require.Len(t, resp.GetContext().GetDnsContext().GetConfigs(), 1)
		require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)

		req.Connection = resp.Clone()

		requireIPv4Lookup(ctx, t, &resolver, "nsc"+fmt.Sprint(i)+".vl3", "10.0.0.1")

		resp, err = nsc.Request(reqCtx, req)
		require.NoError(t, err)

		requireIPv4Lookup(ctx, t, &resolver, "nsc"+fmt.Sprint(i)+".vl3", "10.0.0.1")

		_, err = nsc.Close(reqCtx, resp)
		require.NoError(t, err)

		_, err = resolver.LookupIP(reqCtx, "ip4", "nsc"+fmt.Sprint(i)+".vl3")
		require.Error(t, err)
	}
}

func requireIPv4Lookup(ctx context.Context, t *testing.T, r *net.Resolver, host, expected string) {
	addrs, err := r.LookupIP(ctx, "ip4", host)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	require.Equal(t, expected, addrs[0].String())
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

	var records memory.Map
	var dnsServer = memory.NewDNSHandler(&records)

	records.Store("nsc1.vl3.", []net.IP{net.ParseIP("1.1.1.1")})

	dnsutils.ListenAndServe(ctx, dnsServer, ":40053")

	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)

	nsReg, err := nsRegistryClient.Register(ctx, defaultRegistryService("vl3"))
	require.NoError(t, err)

	nseReg := defaultRegistryEndpoint(nsReg.Name)

	var serverPrefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(serverPrefixCh)

	serverPrefixCh <- &ipam.PrefixResponse{Prefix: "10.0.0.1/24"}

	var dnsConfigs = new(vl3dns.Map)

	_ = domain.Nodes[0].NewEndpoint(
		ctx,
		nseReg,
		sandbox.GenerateTestToken,
		vl3.NewServer(ctx, serverPrefixCh),
		vl3dns.NewServer(ctx,
			func() net.IP { return net.ParseIP("0.0.0.0") },
			vl3dns.WithDomainSchemes("{{ index .Labels \"podName\" }}.{{ .NetworkService }}."),
			vl3dns.WithDNSListenAndServeFunc(func(ctx context.Context, handler dnsutils.Handler, listenOn string) {
				dnsutils.ListenAndServe(ctx, handler, ":50053")
			}),
			vl3dns.WithConfigs(dnsConfigs),
			vl3dns.WithDNSPort(40053),
		),
	)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, "127.0.0.1:50053")
		},
	}

	var clientPrefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(clientPrefixCh)

	clientPrefixCh <- &ipam.PrefixResponse{Prefix: "127.0.0.1/32"}
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken, client.WithAdditionalFunctionality(vl3dns.NewClient(net.ParseIP("127.0.0.1"), dnsConfigs), vl3.NewClient(ctx, clientPrefixCh)))

	req := defaultRequest(nsReg.Name)
	req.Connection.Id = uuid.New().String()

	req.Connection.Labels["podName"] = "nsc"

	resp, err := nsc.Request(ctx, req)
	require.NoError(t, err)
	require.Len(t, resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps, 1)
	require.Equal(t, "127.0.0.1", resp.GetContext().GetDnsContext().GetConfigs()[0].DnsServerIps[0])

	require.Equal(t, "127.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	req.Connection = resp.Clone()

	requireIPv4Lookup(ctx, t, &resolver, "nsc.vl3", "127.0.0.1")

	requireIPv4Lookup(ctx, t, &resolver, "nsc1.vl3", "1.1.1.1") // we can lookup this ip address only and only if fanout is working

	resp, err = nsc.Request(ctx, req)
	require.NoError(t, err)

	requireIPv4Lookup(ctx, t, &resolver, "nsc.vl3", "127.0.0.1")

	requireIPv4Lookup(ctx, t, &resolver, "nsc1.vl3", "1.1.1.1") // we can lookup this ip address only and only if fanout is working

	_, err = nsc.Close(ctx, resp)
	require.NoError(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", "nsc.vl3")
	require.Error(t, err)

	_, err = resolver.LookupIP(ctx, "ip4", "nsc1.vl3")
	require.Error(t, err)
}
