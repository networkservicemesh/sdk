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

// Package dnsconfigs stores dns configs
package dnsconfigs

import (
	"context"
	"net/url"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/tools/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

type dnsConfigsHandler struct {
	configs *Map
}

func (h *dnsConfigsHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	if m == nil {
		dns.HandleFailed(rp, m)
		return
	}

	dnsIPs := make([]url.URL, 0)
	searchDomains := make([]string, 0)

	h.configs.Range(func(key string, value []*networkservice.DNSConfig) bool {
		for _, conf := range value {
			ips := make([]url.URL, len(conf.DnsServerIps))
			for i, ip := range conf.DnsServerIps {
				u, err := url.Parse(ip)
				if err != nil {
					return false
				}
				ips[i] = *u
			}

			dnsIPs = append(dnsIPs, ips...)
			searchDomains = append(searchDomains, conf.SearchDomains...)
		}

		return true
	})

	ctx = dnscontext.WithDNSAddresses(ctx, dnsIPs)
	ctx = dnscontext.WithSearchDomains(ctx, searchDomains)
	next.Handler(ctx).ServeDNS(ctx, rp, m)
}

// NewDNSHandler creates a new dns handler that stores dns configs
func NewDNSHandler(configs *Map) dnsutils.Handler {
	return &dnsConfigsHandler{
		configs: configs,
	}
}