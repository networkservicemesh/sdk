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

// Package searches makes requests to all subdomains received from dns configs
package searches

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	timeout = 5 * time.Second
)

type searchDomainsHandler struct {
	RequestError error
}

func (h *searchDomainsHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	if m == nil {
		dns.HandleFailed(rw, m)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	wrapper := &responseWriterWrapper{
		ResponseWriter: rw,
		handler:        h,
	}

	// TODO: add custom responseWriter to collect all responses from next chain elements and do with them whatever we want
	next.Handler(ctx).ServeDNS(ctx, wrapper, m)

	if h.RequestError == nil {
		return
	}
	log.FromContext(ctx).Warn(h.RequestError)

	for _, d := range SearchDomains(ctx) {
		newMsg := m.Copy()
		newMsg.Question[0].Name = dns.Fqdn(newMsg.Question[0].Name + d)
		next.Handler(ctx).ServeDNS(ctx, wrapper, newMsg)

		if h.RequestError == nil {
			return
		}
		log.FromContext(ctx).Warn(h.RequestError)
	}

	dns.HandleFailed(rw, m)
}

// NewDNSHandler creates a new dns handler that makes requests to all subdomains received from dns configs
func NewDNSHandler() dnsutils.Handler {
	return new(searchDomainsHandler)
}
