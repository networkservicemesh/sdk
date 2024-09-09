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

// Package noloop prevents loops
package noloop

import (
	"context"
	"sync"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type noloopDNSHandler struct{ ids sync.Map }

func (n *noloopDNSHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	if _, loaded := n.ids.LoadOrStore(m.Id, struct{}{}); loaded {
		log.FromContext(ctx).WithField("noloopDNSHandler", "ServeDNS").Errorf("loop is not allowed: query: %v", m.String())
		dns.HandleFailed(rp, m)
		return
	}
	defer n.ids.Delete(m.Id)
	next.Handler(ctx).ServeDNS(ctx, rp, m)
}

// NewDNSHandler creates a new dns handelr that prevents loops.
func NewDNSHandler() dnsutils.Handler {
	return new(noloopDNSHandler)
}
