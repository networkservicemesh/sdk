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

// Package searches makes requests to all subdomains received from DNS configs
package searches

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type searchDomainsHandler struct{}

func (h *searchDomainsHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	for _, d := range append([]string{""}, SearchDomains(ctx)...) {
		r := &responseWriter{
			ResponseWriter: rw,
		}

		newMsg := m.Copy()
		newMsg.Question[0].Name = dns.Fqdn(newMsg.Question[0].Name + d)
		next.Handler(ctx).ServeDNS(ctx, r, newMsg)

		if r.Response != nil && r.Response.Rcode == dns.RcodeSuccess {
			r.Response.Question = m.Question
			if err := rw.WriteMsg(r.Response); err != nil {
				log.FromContext(ctx).WithField("searchDomainsHandler", "ServeDNS").Warnf("got an error during write the message: %v", err.Error())
				dns.HandleFailed(rw, r.Response)
				return
			}
			return
		}
	}

	dns.HandleFailed(rw, m)
}

// NewDNSHandler creates a new dns handler that makes requests to all subdomains received from dns configs.
func NewDNSHandler() dnsutils.Handler {
	return new(searchDomainsHandler)
}
