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

package cache

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestCache(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m := &dns.Msg{}
	m.SetQuestion(dns.Fqdn("example.com"), dns.TypeANY)

	configs := new(dnsconfigs.Map)

	configs.Store("1", []*networkservice.DNSConfig{
		{SearchDomains: []string{}, DnsServerIps: []string{"8.8.8.8"}},
	})

	handler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(configs),
		// NewDNSHandler(),
		fanout.NewDNSHandler(),
	)

	go dnsutils.ListenAndServe(ctx, handler, "127.0.0.1:50053")

	resp1, _ := dns.Exchange(m, "127.0.0.1:50053")
	log.FromContext(ctx).Info(resp1.Answer[0].Header().Ttl)
	time.Sleep(5 * time.Second)
	resp2, _ := dns.Exchange(m, "127.0.0.1:50053")
	log.FromContext(ctx).Info(resp2.Answer[0].Header().Ttl)

	require.Equal(t, resp1.Id, resp2.Id)
	require.NotEqual(t, resp1.Answer[0].Header().Ttl, resp2.Answer[0].Header().Ttl)
}
