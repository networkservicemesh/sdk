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
	"net"

	"github.com/miekg/dns"
)

type responseWriterWrapper struct {
	rw    dns.ResponseWriter
	cache *Map
}

func (r *responseWriterWrapper) LocalAddr() net.Addr {
	return r.rw.LocalAddr()
}

func (r *responseWriterWrapper) RemoteAddr() net.Addr {
	return r.rw.RemoteAddr()
}

func (r *responseWriterWrapper) WriteMsg(m *dns.Msg) error {
	if m.Rcode == 0 {
		r.cache.Store(m.Question[0].Name, m)
	}
	return r.rw.WriteMsg(m)
}

func (r *responseWriterWrapper) Write(p []byte) (int, error) {
	return r.rw.Write(p)
}

func (r *responseWriterWrapper) Close() error {
	return r.rw.Close()
}

func (r *responseWriterWrapper) TsigStatus() error {
	return r.rw.TsigStatus()
}

func (r *responseWriterWrapper) TsigTimersOnly(b bool) {
	r.rw.TsigTimersOnly(b)
}

func (r *responseWriterWrapper) Hijack() {
	r.rw.Hijack()
}
