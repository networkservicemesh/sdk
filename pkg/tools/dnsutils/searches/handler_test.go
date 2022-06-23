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

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/fanout"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
)

func TestDomainSearches(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	configs := new(dnsconfigs.Map)

	configs.Store("1", []*networkservice.DNSConfig{
		{SearchDomains: []string{"com", "net", "org"}, DnsServerIps: []string{"8.8.4.4"}},
	})

	handler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(configs),
		searches.NewDNSHandler(),
		fanout.NewDNSHandler(),
	)

	go dnsutils.ListenAndServe(ctx, handler, "127.0.0.1:40053")

	client := dns.Client{
		Net:     "udp",
		Timeout: time.Hour,
	}
	m1 := &dns.Msg{}
	m1.SetQuestion(dns.Fqdn("example"), dns.TypeANY)
	resp, _, err := client.Exchange(m1, "127.0.0.1:40053")
	require.NoError(t, err)
	require.Equal(t, resp.MsgHdr.Rcode, 0)
	require.True(t, strings.HasSuffix(resp.Question[0].Name, ".com."))
}
