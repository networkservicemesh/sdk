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

package searches_test

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
)

type responseWriter struct {
	dns.ResponseWriter
	Response *dns.Msg
}

func (r *responseWriter) WriteMsg(m *dns.Msg) error {
	r.Response = m
	return nil
}

type checkHandler struct {
	Count int
}

func (h *checkHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	h.Count++
	next.Handler(ctx).ServeDNS(ctx, rw, m)
}

func TestDomainSearches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	configs := new(dnsconfig.Map)

	configs.Store("1", []*networkservice.DNSConfig{
		{SearchDomains: []string{"com", "net", "org"}, DnsServerIps: []string{"8.8.4.4"}},
	})

	records := new(memory.Map)
	records.Store("example.com.", []net.IP{net.ParseIP("1.1.1.1")})

	check := &checkHandler{}
	handler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(configs),
		searches.NewDNSHandler(),
		check,
		memory.NewDNSHandler(records),
	)

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn("example"), dns.TypeA)

	rw := &responseWriter{}
	handler.ServeDNS(ctx, rw, m)

	resp := rw.Response.Copy()
	require.Equal(t, check.Count, 4)
	require.Equal(t, resp.MsgHdr.Rcode, dns.RcodeSuccess)
	require.NotNil(t, resp.Answer)
	require.Equal(t, resp.Answer[0].(*dns.A).A.String(), "1.1.1.1")
}
