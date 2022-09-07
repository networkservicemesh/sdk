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

// Package searches makes requests to all subdomains received from DNS configs
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
}

func (h *searchDomainsHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	domains := SearchDomains(ctx)

	r := &responseWriter{
		ResponseWriter: rw,
		Responses:      make([]*dns.Msg, len(domains)+1),
		index:          0,
	}

	// We need to use a split timeout context, because a request to one of the domains
	// may take all the available time, and it will not be possible to request the others.
	timeoutPerRequest := timeout / time.Duration(len(domains)+1)

	// If the context already has timeout
	if deadline, ok := ctx.Deadline(); ok {
		t := time.Until(deadline)
		timeoutPerRequest = t / time.Duration(len(domains)+1)
	}

	ctxPerRequest, cancelCtxPerRequest := context.WithTimeout(ctx, timeoutPerRequest)
	next.Handler(ctx).ServeDNS(ctxPerRequest, r, m)
	cancelCtxPerRequest()

	for _, d := range domains {
		ctxPerRequest, cancelCtxPerRequest = context.WithTimeout(ctx, timeoutPerRequest)

		newMsg := m.Copy()
		newMsg.Question[0].Name = dns.Fqdn(newMsg.Question[0].Name + d)
		next.Handler(ctx).ServeDNS(ctxPerRequest, r, newMsg)

		cancelCtxPerRequest()
	}

	for _, resp := range r.Responses {
		if resp != nil && resp.Rcode == dns.RcodeSuccess {
			resp.Question = m.Question
			if err := rw.WriteMsg(resp); err != nil {
				log.FromContext(ctx).WithField("searchDomainsHandler", "ServeDNS").Warnf("got an error during write the message: %v", err.Error())
				dns.HandleFailed(rw, resp)
				return
			}
			return
		}
	}

	dns.HandleFailed(rw, m)
}

// NewDNSHandler creates a new dns handler that makes requests to all subdomains received from dns configs
func NewDNSHandler() dnsutils.Handler {
	return new(searchDomainsHandler)
}
