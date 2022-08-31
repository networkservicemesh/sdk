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

// Package cache stores successful requests to DNS server
package cache

import (
	"context"
	"sync"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type dnsCacheHandler struct {
	cache *msgMap

	lastTTLUpdate time.Time
	m             sync.Mutex
}

func (h *dnsCacheHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, m *dns.Msg) {
	h.updateTTL()
	if val, ok := h.cache.Load(m.Question[0]); ok {
		v := val.Copy()
		if validateMsg(v) {
			v.Id = m.Id
			if err := rw.WriteMsg(v); err != nil {
				log.FromContext(ctx).WithField("dnsCacheHandler", "ServeDNS").Warnf("got an error during write the message: %v", err.Error())
				dns.HandleFailed(rw, v)
				return
			}
			return
		}

		h.cache.Delete(m.Question[0])
	}

	wrapper := responseWriterWrapper{
		ResponseWriter: rw,
		cache:          h.cache,
	}

	next.Handler(ctx).ServeDNS(ctx, &wrapper, m)
}

func (h *dnsCacheHandler) updateTTL() {
	now := time.Now()
	h.m.Lock()
	defer h.m.Unlock()

	diff := uint32(now.Sub(h.lastTTLUpdate).Seconds())
	if diff == 0 {
		return
	}

	h.cache.Range(func(key dns.Question, value *dns.Msg) bool {
		for i := range value.Answer {
			if value.Answer[i].Header().Ttl < diff {
				value.Answer[i].Header().Ttl = 0
			} else {
				value.Answer[i].Header().Ttl -= diff
			}
		}
		return true
	})
	h.lastTTLUpdate = now
}

func validateMsg(m *dns.Msg) bool {
	if len(m.Answer) == 0 {
		return false
	}
	for _, answer := range m.Answer {
		if answer.Header().Ttl <= 0 {
			return false
		}
	}

	return true
}

// NewDNSHandler creates a new dns handler that stores successful requests to DNS server
func NewDNSHandler() dnsutils.Handler {
	return &dnsCacheHandler{
		cache:         new(msgMap),
		lastTTLUpdate: time.Now(),
	}
}
