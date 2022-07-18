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

// Package memory provides a/aaaa memory storage
package memory

import (
	"context"
	"net"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

const defaultTTL = 3600

type memoryHandler struct {
	recoreds *Map
}

func (f *memoryHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	if len(msg.Question) == 0 {
		next.Handler(ctx).ServeDNS(ctx, rw, msg)
		return
	}

	var name = dns.Name(msg.Question[0].Name).String()
	var records, ok = f.recoreds.Load(name)

	if !ok {
		next.Handler(ctx).ServeDNS(ctx, rw, msg)
		return
	}

	var resp = new(dns.Msg)
	resp.SetReply(msg)
	resp.Authoritative = true

	switch msg.Question[0].Qtype {
	case dns.TypeAAAA:
		resp.Answer = append(resp.Answer, aaaa(name, records)...)
	case dns.TypeA:
		resp.Answer = append(resp.Answer, a(name, records)...)
	}

	if len(resp.Answer) == 0 {
		next.Handler(ctx).ServeDNS(ctx, rw, msg)
		return
	}

	if err := rw.WriteMsg(resp); err != nil {
		dns.HandleFailed(rw, msg)
	}
}

// NewDNSHandler creates a new dns handler instance that stores a/aaaa answers
func NewDNSHandler(records *Map) dnsutils.Handler {
	if records == nil {
		panic("records cannot be nil")
	}
	return &memoryHandler{recoreds: records}
}
func a(domain string, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		if ip.To4() == nil {
			continue
		}
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: defaultTTL}
		r.A = ip
		answers[i] = r
	}
	return answers
}

func aaaa(domain string, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		if ip.To16() == nil {
			continue
		}
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: defaultTTL}
		r.AAAA = ip
		answers[i] = r
	}
	return answers
}
