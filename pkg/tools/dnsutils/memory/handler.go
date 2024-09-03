// Copyright (c) 2022-2023 Cisco Systems, Inc.
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
	"strings"

	"github.com/edwarnicke/genericsync"
	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

const defaultTTL = 3600

// Since memory is supposed to be one of the targets that stores information, we have to keep track of whether something has been written to the writer.
// We must write something into the writer, this is how the dns package works. Otherwise, we get a timeout error on the client side.

type responseWriter struct {
	dns.ResponseWriter
	passed bool
}

func (r *responseWriter) WriteMsg(m *dns.Msg) error {
	r.passed = true
	return r.ResponseWriter.WriteMsg(m)
}

type memoryHandler struct {
	records *genericsync.Map[string, []net.IP]
}

func (f *memoryHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	if len(msg.Question) == 0 {
		dns.HandleFailed(rw, msg)
		return
	}

	name := dns.Name(msg.Question[0].Name).String()
	resp := new(dns.Msg)
	resp.SetReply(msg)
	resp.Authoritative = true

	switch msg.Question[0].Qtype {
	case dns.TypeAAAA:
		resp.Answer = append(resp.Answer, f.aaaa(name)...)
	case dns.TypeA:
		resp.Answer = append(resp.Answer, f.a(name)...)
	case dns.TypePTR:
		resp.Answer = append(resp.Answer, f.ptr(name)...)
	}

	if len(resp.Answer) != 0 {
		if err := rw.WriteMsg(resp); err != nil {
			dns.HandleFailed(rw, msg)
		}
		return
	}

	if _, ok := f.records.Load(name); ok {
		m := new(dns.Msg)
		_ = rw.WriteMsg(m.SetRcode(msg, dns.RcodeSuccess))
	} else {
		rwWrapper := &responseWriter{ResponseWriter: rw}
		next.Handler(ctx).ServeDNS(ctx, rwWrapper, msg)

		if !rwWrapper.passed {
			dns.HandleFailed(rw, msg)
		}
	}
}

// NewDNSHandler creates a new dns handler instance that stores a/aaaa answers.
func NewDNSHandler(records *genericsync.Map[string, []net.IP]) dnsutils.Handler {
	if records == nil {
		panic("records cannot be nil")
	}
	return &memoryHandler{records: records}
}

func (f *memoryHandler) a(domain string) []dns.RR {
	ips, _ := f.records.Load(domain)
	var answers []dns.RR
	for _, ip := range ips {
		if ip.To4() == nil {
			continue
		}
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: defaultTTL}
		r.A = ip
		answers = append(answers, r)
	}
	return answers
}

func (f *memoryHandler) aaaa(domain string) []dns.RR {
	ips, _ := f.records.Load(domain)
	var answers []dns.RR
	for _, ip := range ips {
		if ip.To4() != nil {
			continue
		}
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: defaultTTL}
		r.AAAA = ip
		answers = append(answers, r)
	}
	return answers
}

func (f *memoryHandler) ptr(domain string) []dns.RR {
	var answers []dns.RR
	var ipArrayStr []string
	if strings.HasSuffix(domain, ".in-addr.arpa.") {
		// IPv4
		ipArrayStr = strings.Split(domain, ".")[:4]
	} else if strings.HasSuffix(domain, ".ip6.arpa.") {
		// IPv6
		ipArrayStr = strings.Split(domain, ".")[:32]
	}

	if len(ipArrayStr) != 0 {
		ipArrayStr = reverse(ipArrayStr)
		requestedIP := net.ParseIP(strings.Join(ipArrayStr, "."))
		if len(ipArrayStr) > 4 {
			// join IPv6 address in groups of 4
			sb := strings.Builder{}
			for i := 0; i < len(ipArrayStr); i++ {
				if i%4 == 0 && i != 0 {
					sb.WriteByte(':')
				}
				sb.WriteString(ipArrayStr[i])
			}
			requestedIP = net.ParseIP(sb.String())
		}

		var recordNames []string
		f.records.Range(func(key string, value []net.IP) bool {
			for _, v := range value {
				if v.Equal(requestedIP) {
					recordNames = append(recordNames, key)
					return true
				}
			}
			return true
		})

		for _, recordName := range recordNames {
			r := new(dns.PTR)
			r.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: defaultTTL}
			r.Ptr = recordName
			answers = append(answers, r)
		}
	}
	return answers
}

func reverse(ss []string) []string {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return ss
}
