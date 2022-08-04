// Copyright (c) 2022 Cisco Systems, Inc.
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

// Package next allows to dns handlers be joined into chain
package next

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
)

type nextDNSHandler struct {
	handlers   []dnsutils.Handler
	index      int
	nextParent dnsutils.Handler
}

// ServerWrapper - A function that wraps a dnsutils.Handler
type ServerWrapper func(dnsutils.Handler) dnsutils.Handler

// NewWrappedDNSHandler - chains together the servers provides with the wrapper wrapped around each one in turn.
func NewWrappedDNSHandler(wrapper ServerWrapper, handlers ...dnsutils.Handler) dnsutils.Handler {
	rv := &nextDNSHandler{handlers: make([]dnsutils.Handler, 0, len(handlers))}
	for _, h := range handlers {
		rv.handlers = append(rv.handlers, wrapper(h))
	}
	return rv
}

// NewDNSHandler creates a new chain of dns handlers
func NewDNSHandler(handlers ...dnsutils.Handler) dnsutils.Handler {
	return &nextDNSHandler{handlers: handlers}
}

func (n *nextDNSHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	nextHandler, nextCtx := n.getClientAndContext(ctx)
	nextHandler.ServeDNS(nextCtx, rp, m)
}

func (n *nextDNSHandler) getClientAndContext(ctx context.Context) (dnsutils.Handler, context.Context) {
	nextParent := n.nextParent
	if n.index == 0 {
		nextParent = Handler(ctx)
		if len(n.handlers) == 0 {
			return nextParent, ctx
		}
	}
	if n.index+1 < len(n.handlers) {
		return n.handlers[n.index], withNextHandler(ctx, &nextDNSHandler{nextParent: nextParent, handlers: n.handlers, index: n.index + 1})
	}
	return n.handlers[n.index], withNextHandler(ctx, nextParent)
}
