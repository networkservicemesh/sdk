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
	"net/url"
	"strings"
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
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/noloop"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
)

const (
	resolvedIP      = "1.1.1.1"
	healthCheckHost = "health.check.only"
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

type tcpHandler struct {
}

func (h *tcpHandler) ServeDNS(rw dns.ResponseWriter, m *dns.Msg) {
	time.Sleep(time.Second * 5)
	dns.HandleFailed(rw, m)
}

type udpHandler struct {
}

func (h *udpHandler) ServeDNS(rw dns.ResponseWriter, m *dns.Msg) {
	name := m.Question[0].Name

	if name == ToWildcardPath(healthCheckHost) {
		resp := prepareIPResponse(name, resolvedIP, m)
		if err := rw.WriteMsg(resp); err != nil {
			dns.HandleFailed(rw, m)
		}
		return
	}

	if name == "my.domain." {
		dns.HandleFailed(rw, m)
		return
	}

	resp := prepareIPResponse(name, resolvedIP, m)
	if err := rw.WriteMsg(resp); err != nil {
		dns.HandleFailed(rw, m)
	}
}

func prepareIPResponse(name, ip string, m *dns.Msg) *dns.Msg {
	dnsName := dns.Name(name).String()
	rr := new(dns.A)
	rr.Hdr = dns.RR_Header{Name: dnsName, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600}
	rr.A = net.ParseIP(ip)

	resp := new(dns.Msg)
	resp.SetReply(m)
	resp.Authoritative = true
	resp.Answer = append(resp.Answer, rr)

	return resp
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

	healthCheckParams := &HealthCheckParams{
		DNSServerIP: proxyAddr,
		HealthHost:  healthCheckHost,
		Scheme:      "udp",
	}

	clientDNSHandler := next.NewDNSHandler(
		NewTestDNSHandler(*healthCheckParams, []TestDNSOption{}...),
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
		addrs, e := resolver.LookupIP(ctx, "ip4", healthCheckHost)
		return e == nil && len(addrs) == 1 && addrs[0].String() == resolvedIP
	}
	require.Eventually(t, healthCheck, time.Second, 10*time.Millisecond)

	resolveCtx, resolveCancel := context.WithTimeout(context.Background(), time.Second*5)
	defer resolveCancel()

	requireIPv4Lookup(resolveCtx, t, &resolver, "my.domain", resolvedIP)

	err = proxy.shutdown()
	require.NoError(t, err)
}

// HealthCheckParams required parameters for the health check handler
type HealthCheckParams struct {
	// DNSServerIP provides the IP address of the DNS server that will be checked for connectivity
	DNSServerIP string
	// HealthHost value used to check the DNS server
	// DNS server should resolve it correctly
	// Make sure that you properly configure server first before using this handler
	// Note: All other requests that do not match the HealthPath will be forwarded without any changes or actions
	HealthHost string
	// Scheme "tcp" or "udp" connection type
	Scheme string
}

type dnsHealthCheckHandler struct {
	dnsServerIP string
	healthHost  string
	scheme      string
	dnsPort     uint16
}

func (h *dnsHealthCheckHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	name := msg.Question[0].Name

	if name != h.healthHost {
		next.Handler(ctx).ServeDNS(ctx, rw, msg)
		return
	}

	newMsg := msg.Copy()
	newMsg.Question[0].Name = dns.Fqdn(name)
	dnsIP := url.URL{Scheme: h.scheme, Host: h.dnsServerIP}

	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)

	var responseCh = make(chan *dns.Msg)

	go func(u *url.URL, msg *dns.Msg) {
		var client = dns.Client{
			Net:     u.Scheme,
			Timeout: timeout,
		}

		// If u.Host is IPv6 then wrap it in brackets
		if strings.Count(u.Host, ":") >= 2 && !strings.HasPrefix(u.Host, "[") && !strings.Contains(u.Host, "]") {
			u.Host = fmt.Sprintf("[%s]", u.Host)
		}

		address := u.Host
		if u.Port() == "" {
			address += fmt.Sprintf(":%d", h.dnsPort)
		}

		var resp, _, err = client.Exchange(msg, address)
		if err != nil {
			responseCh <- nil
			return
		}

		responseCh <- resp
	}(&dnsIP, newMsg.Copy())

	var resp = h.waitResponse(ctx, responseCh)

	if resp == nil {
		dns.HandleFailed(rw, newMsg)
		return
	}

	if err := rw.WriteMsg(resp); err != nil {
		dns.HandleFailed(rw, newMsg)
		return
	}
}

func (h *dnsHealthCheckHandler) waitResponse(ctx context.Context, respCh <-chan *dns.Msg) *dns.Msg {
	var respCount = cap(respCh)
	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				return nil
			}
			respCount--
			if resp == nil {
				if respCount == 0 {
					return nil
				}
				continue
			}
			if resp.Rcode == dns.RcodeSuccess {
				return resp
			}
			if respCount == 0 {
				return nil
			}

		case <-ctx.Done():
			return nil
		}
	}
}

// NewTestDNSHandler creates a new health check dns handler
// dnsHealthCheckHandler is expected to be placed at the beginning of the handlers chain
func NewTestDNSHandler(params HealthCheckParams, opts ...TestDNSOption) dnsutils.Handler {
	var h = &dnsHealthCheckHandler{
		dnsServerIP: params.DNSServerIP,
		healthHost:  ToWildcardPath(params.HealthHost),
		scheme:      params.Scheme,
		dnsPort:     53,
	}

	for _, o := range opts {
		o(h)
	}
	return h
}

// ToWildcardPath will modify host by adding dot at the end
func ToWildcardPath(host string) string {
	return fmt.Sprintf("%s.", host)
}

// Option modifies default health check dns handler values
type TestDNSOption func(*dnsHealthCheckHandler)

// WithDefaultDNSPort sets default DNS port for health check dns handler if it is not presented in the client's URL
func WithDefaultDNSPort(port uint16) TestDNSOption {
	return func(h *dnsHealthCheckHandler) {
		h.dnsPort = port
	}
}
