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

// Package dnsutils provides dns specific utils functions and packages
package dnsutils

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// ListenAndServe starts dns server with specific handler. Listens both udp/tcp networks.
// ctx is using for keeping the server alive. As soon as <-ctx.Done() happens it stops dns server.
// handler is using for hanlding dns queries.
// listenOn is using for listen. Expects {ip}:{port} to listen. Examples: "127.0.0.1:53", ":53".
func ListenAndServe(ctx context.Context, handler Handler, listenOn string) {
	var networks = []string{"tcp", "udp"}

	for _, network := range networks {
		var server = &dns.Server{Addr: listenOn, Net: network, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, m *dns.Msg) {
			var timeoutCtx, cancel = context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()

			handler.ServeDNS(timeoutCtx, w, m)
		})}

		go func() {
			<-ctx.Done()
			_ = server.Shutdown()
		}()

		go func() {
			for ; ctx.Err() == nil; time.Sleep(time.Millisecond / 100) {
				var err = server.ListenAndServe()
				if err != nil {
					log.FromContext(ctx).Errorf("an error during serve dns: %v", err.Error())
				}
			}
		}()
	}
}

// ContainsDNSConfig returns true if array contains a specific dns config
func ContainsDNSConfig(array []*networkservice.DNSConfig, value *networkservice.DNSConfig) bool {
	for i := range array {
		if equal(array[i].DnsServerIps, value.DnsServerIps) && equal(array[i].SearchDomains, value.SearchDomains) {
			return true
		}
	}
	return false
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	diff := make(map[string]int, len(a))
	for _, v := range a {
		diff[v]++
	}

	for _, v := range b {
		if _, ok := diff[v]; !ok {
			return false
		}
		diff[v]--
		if diff[v] == 0 {
			delete(diff, v)
		}
	}
	return len(diff) == 0
}
