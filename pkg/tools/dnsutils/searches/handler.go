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

// Package searches ???
package searches

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnscontext"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

type dnsConfigsHandler struct {
}

func (n *dnsConfigsHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	if m == nil {
		dns.HandleFailed(rp, m)
		return
	}

	searchDomains := dnscontext.SearchDomains(ctx)

	for _, d := range searchDomains {
		newMsg := m.Copy()
		newMsg.Question[0].Name += d + "."
		next.Handler(ctx).ServeDNS(ctx, rp, newMsg)
	}

	next.Handler(ctx).ServeDNS(ctx, rp, m)
}

// NewDNSHandler creates a new dns handler that stores dns configs
func NewDNSHandler() dnsutils.Handler {
	return new(dnsConfigsHandler)
}
