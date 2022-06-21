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
package cache

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

type dnsCacheHandler struct {
	cache         *Map
	lastTTLUpdate time.Time
}

func (h *dnsCacheHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	if m == nil {
		dns.HandleFailed(rp, m)
		return
	}

	h.updateTTL()

	// падаем если хотя бы 1 отрицательный Answer
	if v, ok := h.cache.Load(m.Question[0].Name); ok {
		if v.Answer[0].Header().Ttl > 0 {
			rp.WriteMsg(v)
			return
		}

		h.cache.Delete(m.Question[0].Name)
	}

	w := responseWriterWrapper{
		rw:    rp,
		cache: h.cache,
	}

	next.Handler(ctx).ServeDNS(ctx, &w, m)
}

func (h *dnsCacheHandler) updateTTL() {
	now := time.Now()
	h.cache.Range(func(key string, value *dns.Msg) bool {
		sub := now.Sub(h.lastTTLUpdate).Seconds()
		subint := uint32(sub)
		value.Answer[0].Header().Ttl -= subint
		return true
	})
	h.lastTTLUpdate = now
}

// NewDNSHandler creates a new dns handler that stores dns configs
func NewDNSHandler() dnsutils.Handler {
	return &dnsCacheHandler{
		cache: new(Map),
	}
}
