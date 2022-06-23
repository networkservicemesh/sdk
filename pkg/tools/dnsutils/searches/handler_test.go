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
	"strings"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ResponseWriter struct {
	dns.ResponseWriter
	Response *dns.Msg
}

func (r *ResponseWriter) WriteMsg(m *dns.Msg) error {
	r.Response = m
	return nil
}

type checkHandler struct {
	Count int
}

func (h *checkHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	h.Count++
	var err error
	if m.Question[0].Name == "example." {
		resp := new(dns.Msg)
		resp.Rcode = 2
		err = rw.WriteMsg(resp)
	} else {
		err = rw.WriteMsg(m)
	}

	if err != nil {
		log.FromContext(ctx).Error(err)
	}
}

func TestDomainSearches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	configs := new(dnsconfigs.Map)

	configs.Store("1", []*networkservice.DNSConfig{
		{SearchDomains: []string{"com", "net", "org"}, DnsServerIps: []string{"8.8.4.4"}},
	})

	check := &checkHandler{}
	handler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(configs),
		searches.NewDNSHandler(),
		check,
	)

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn("example"), dns.TypeANY)

	rw := &ResponseWriter{}
	handler.ServeDNS(ctx, rw, m)

	resp := rw.Response.Copy()
	require.Equal(t, check.Count, 4)
	require.Equal(t, resp.MsgHdr.Rcode, 0)
	require.True(t, strings.HasSuffix(resp.Question[0].Name, ".com."))
}
