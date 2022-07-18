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

package cache_test

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/cache"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
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
	next.Handler(ctx).ServeDNS(ctx, rw, m)
}

func TestCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	records := new(memory.Map)
	records.Store("example.com.", []net.IP{net.ParseIP("1.1.1.1")})

	check := &checkHandler{}
	handler := next.NewDNSHandler(
		cache.NewDNSHandler(),
		check,
		memory.NewDNSHandler(records),
	)

	rw := &ResponseWriter{}
	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)
	handler.ServeDNS(ctx, rw, m)
	resp1 := rw.Response.Copy()

	time.Sleep(time.Second)

	handler.ServeDNS(ctx, rw, m)
	resp2 := rw.Response.Copy()

	require.Equal(t, check.Count, 1)
	require.Equal(t, resp1.Answer[0].Header().Ttl-resp2.Answer[0].Header().Ttl, uint32(1))
}
