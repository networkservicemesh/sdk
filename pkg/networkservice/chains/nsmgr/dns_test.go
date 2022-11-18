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

//go:build !windows
// +build !windows

package nsmgr_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
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

	dnsConfigsMap := new(dnsconfig.Map)
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
		fanout.NewDNSHandler(),
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

type proxyDNSServer struct {
	ListenOn  string
	TCPServer *dns.Server
	UDPServer *dns.Server
}

func (p *proxyDNSServer) listenAndServe(t *testing.T, tcpHandler, udpHandler dns.Handler) {
	p.TCPServer = &dns.Server{Addr: p.ListenOn, Net: "tcp", Handler: tcpHandler}
	p.UDPServer = &dns.Server{Addr: p.ListenOn, Net: "udp", Handler: udpHandler}

	go func() {
		err := p.TCPServer.ListenAndServe()
		require.NoError(t, err)
	}()

	go func() {
		err := p.UDPServer.ListenAndServe()
		require.NoError(t, err)
	}()
}

func (p *proxyDNSServer) shutdown() error {
	tcpErr := p.TCPServer.Shutdown()
	udpErr := p.UDPServer.Shutdown()

	if tcpErr != nil {
		return tcpErr
	}

	if udpErr != nil {
		return udpErr
	}

	return nil
}

type tcpHandler struct {
}

func (h *tcpHandler) ServeDNS(rw dns.ResponseWriter, m *dns.Msg) {
	time.Sleep(time.Second * 2)
	dns.HandleFailed(rw, m)
}

type udpHandler struct {
}

func (h *udpHandler) ServeDNS(rw dns.ResponseWriter, m *dns.Msg) {
	if m.Question[0].Name == "my.domain." {
		dns.HandleFailed(rw, m)
		return
	}

	name := dns.Name(m.Question[0].Name).String()
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600}
	rr.A = net.ParseIP("1.1.1.1")

	resp := new(dns.Msg)
	resp.SetReply(m)
	resp.Authoritative = true
	resp.Answer = append(resp.Answer, rr)

	if err := rw.WriteMsg(resp); err != nil {
		dns.HandleFailed(rw, m)
	}
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	err = l.Close()
	if err != nil {
		return 0, err
	}

	return l.Addr().(*net.TCPAddr).Port, nil
}

func Test_TCPDNSServerTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	proxyPort, err := getFreePort()
	require.NoError(t, err)
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)

	proxy := &proxyDNSServer{ListenOn: proxyAddr}
	proxy.listenAndServe(t, &tcpHandler{}, &udpHandler{})

	clientPort, err := getFreePort()
	require.NoError(t, err)
	clientAddr := fmt.Sprintf("127.0.0.1:%d", clientPort)

	dnsConfigsMap := new(dnsconfig.Map)
	dnsConfigsMap.Store("1", []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{proxyAddr},
			SearchDomains: []string{"com"},
		},
	})

	clientDNSHandler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(dnsConfigsMap),
		searches.NewDNSHandler(),
		noloop.NewDNSHandler(),
		fanout.NewDNSHandler(),
	)
	dnsutils.ListenAndServe(ctx, clientDNSHandler, clientAddr)

	resolver := net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, clientAddr)
		},
	}

	requireIPv4Lookup(ctx, t, &resolver, "my.domain", "1.1.1.1")

	err = proxy.shutdown()
	require.NoError(t, err)
}
