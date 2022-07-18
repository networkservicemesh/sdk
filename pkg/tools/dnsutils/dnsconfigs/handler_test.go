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

package dnsconfigs_test

import (
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/searches"
)

type checkHandler struct {
	Domains []string
	URLs    []string
}

func (h *checkHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	h.Domains = searches.SearchDomains(ctx)

	urls := clienturlctx.ClientURLs(ctx)

	for i := range urls {
		h.URLs = append(h.URLs, urls[i].String())
	}
}

func TestDNSConfigs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	configs := new(dnsconfig.Map)

	configs.Store("1", []*networkservice.DNSConfig{
		{
			SearchDomains: []string{"example.com"},
			DnsServerIps:  []string{"7.7.7.7"},
		},
		{
			SearchDomains: []string{"net"},
			DnsServerIps:  []string{"1.1.1.1"},
		},
	})

	configs.Store("2", []*networkservice.DNSConfig{
		{
			SearchDomains: []string{"my.domain"},
			DnsServerIps:  []string{"9.9.9.9"},
		},
	})

	check := &checkHandler{}
	handler := next.NewDNSHandler(
		dnsconfigs.NewDNSHandler(configs),
		check,
	)

	handler.ServeDNS(ctx, nil, new(dns.Msg))

	domains := check.Domains
	require.Equal(t, len(domains), 3)
	require.Contains(t, domains, "example.com")
	require.Contains(t, domains, "my.domain")
	require.Contains(t, domains, "net")

	urls := check.URLs
	require.Equal(t, len(urls), 3)
	require.Contains(t, urls, "tcp://7.7.7.7")
	require.Contains(t, urls, "tcp://1.1.1.1")
	require.Contains(t, urls, "tcp://9.9.9.9")
}
