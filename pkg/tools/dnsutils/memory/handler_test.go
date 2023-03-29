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

package memory_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/edwarnicke/genericsync"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

type responseWriter struct {
	dns.ResponseWriter
	Response *dns.Msg
}

func (r *responseWriter) WriteMsg(m *dns.Msg) error {
	r.Response = m
	return nil
}

func Test_A_AAAA(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Store two entries for IPv4 and IPv6
	records := new(genericsync.Map[string, []net.IP])
	records.Store("example.com.", []net.IP{net.ParseIP("1.1.1.1")})
	records.Store("example.net.", []net.IP{net.ParseIP("2001:db8::68")})

	handler := next.NewDNSHandler(
		memory.NewDNSHandler(records),
	)
	rw := &responseWriter{}
	m := &dns.Msg{}

	// Get example.com IPv4. Expect success
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	handler.ServeDNS(ctx, rw, m)

	resp := rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.NotNil(t, resp.Answer)
	require.Equal(t, resp.Answer[0].(*dns.A).A.String(), "1.1.1.1")

	// Get example.com IPv6. Expect NXDomain
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeAAAA)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.Len(t, resp.Answer, 0)

	// Get example.net IPv4. Expect NXDomain
	m.SetQuestion(dns.Fqdn("example.net"), dns.TypeA)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.Len(t, resp.Answer, 0)

	// Get example.net IPv6. Expect success
	m.SetQuestion(dns.Fqdn("example.net"), dns.TypeAAAA)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.NotNil(t, resp.Answer)
	require.Equal(t, resp.Answer[0].(*dns.AAAA).AAAA.String(), "2001:db8::68")
}

func Test_PTR(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Store two entries for IPv4 and IPv6
	records := new(genericsync.Map[string, []net.IP])
	records.Store("example.com.", []net.IP{net.ParseIP("1.1.1.2")})
	records.Store("example.net.", []net.IP{net.ParseIP("2001:db8::68")})

	handler := next.NewDNSHandler(
		memory.NewDNSHandler(records),
	)
	rw := &responseWriter{}
	m := &dns.Msg{}

	// PTR IPv4. Expect success
	m.SetQuestion(dns.Fqdn("2.1.1.1.in-addr.arpa"), dns.TypePTR)
	handler.ServeDNS(ctx, rw, m)

	resp := rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.NotNil(t, resp.Answer)
	require.Equal(t, resp.Answer[0].(*dns.PTR).Ptr, "example.com.")

	// PTR IPv6. Expect success
	m.SetQuestion(dns.Fqdn("8.6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."), dns.TypePTR)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.NotNil(t, resp.Answer)
	require.Equal(t, resp.Answer[0].(*dns.PTR).Ptr, "example.net.")

	// PTR IPv4. Expect fail
	m.SetQuestion(dns.Fqdn("3.1.1.1.in-addr.arpa"), dns.TypePTR)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeServerFailure)

	// PTR IPv6. Expect fail
	m.SetQuestion(dns.Fqdn("9.6.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."), dns.TypePTR)
	handler.ServeDNS(ctx, rw, m)

	resp = rw.Response.Copy()
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeServerFailure)
}
