// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package dnsconfigs_test

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
)

func requireIPv4Lookup(ctx context.Context, t *testing.T, r *net.Resolver, host, expected string) {
	addrs, err := r.LookupIP(ctx, "ip4", host)
	require.NoError(t, err)
	require.Len(t, addrs, 1)
	require.Equal(t, expected, addrs[0].String())
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

type tcpHandler struct{}

func (h *tcpHandler) ServeDNS(rw dns.ResponseWriter, m *dns.Msg) {
	time.Sleep(time.Second * 5)
	dns.HandleFailed(rw, m)
}

type udpHandler struct{}

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

	dnsConfigsMap := new(genericsync.Map[string, []*networkservice.DNSConfig])
	dnsConfigsMap.Store("1", []*networkservice.DNSConfig{
		{
			DnsServerIps:  []string{proxyAddr},
			SearchDomains: []string{"com"},
		},
	})

	dnsServerRecords := new(genericsync.Map[string, []net.IP])
	dnsServerRecords.Store("health.check.only.", []net.IP{net.ParseIP("1.0.0.1")})

	clientDNSHandler := next.NewDNSHandler(
		memory.NewDNSHandler(dnsServerRecords),
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

	healthCheck := func() bool {
		addrs, e := resolver.LookupIP(ctx, "ip4", "health.check.only")
		return e == nil && len(addrs) == 1 && addrs[0].String() == "1.0.0.1"
	}
	require.Eventually(t, healthCheck, time.Second, 10*time.Millisecond)

	resolveCtx, resolveCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer resolveCancel()

	requireIPv4Lookup(resolveCtx, t, &resolver, "my.domain", "1.1.1.1")

	err = proxy.shutdown()
	require.NoError(t, err)
}
