// Copyright (c) 2023 Doc.ai and/or its affiliates.
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

// Package health provides handler to check if server is running
package health

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

// Params required parameters for the health check handler
type Params struct {
	// DNSServerIP provides the IP address of the DNS server that will be checked for connectivity
	DNSServerIP string
	// HealthHost value used to check the DNS server
	// DNS server should resolve it correctly
	// Make sure that you properly configure server first before using this handler
	// Note: All other requests that do not match the HealthPath will be forwarded without any changes or actions
	HealthHost string
	// Scheme "tcp" or "udp" connection type
	Scheme string
}

type dnsHealthCheckHandler struct {
	dnsServerIP string
	healthHost  string
	scheme      string
	dnsPort     uint16
}

func (h *dnsHealthCheckHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	name := msg.Question[0].Name

	if name != h.healthHost {
		next.Handler(ctx).ServeDNS(ctx, rw, msg)
		return
	}

	newMsg := msg.Copy()
	newMsg.Question[0].Name = dns.Fqdn(name)
	dnsIP := url.URL{Scheme: h.scheme, Host: h.dnsServerIP}

	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)

	var responseCh = make(chan *dns.Msg)

	go func(u *url.URL, msg *dns.Msg) {
		var client = dns.Client{
			Net:     u.Scheme,
			Timeout: timeout,
		}

		// If u.Host is IPv6 then wrap it in brackets
		if strings.Count(u.Host, ":") >= 2 && !strings.HasPrefix(u.Host, "[") && !strings.Contains(u.Host, "]") {
			u.Host = fmt.Sprintf("[%s]", u.Host)
		}

		address := u.Host
		if u.Port() == "" {
			address += fmt.Sprintf(":%d", h.dnsPort)
		}

		var resp, _, err = client.Exchange(msg, address)
		if err != nil {
			responseCh <- nil
			return
		}

		responseCh <- resp
	}(&dnsIP, newMsg.Copy())

	var resp = h.waitResponse(ctx, responseCh)

	if resp == nil {
		dns.HandleFailed(rw, newMsg)
		return
	}

	if err := rw.WriteMsg(resp); err != nil {
		dns.HandleFailed(rw, newMsg)
		return
	}
}

func (h *dnsHealthCheckHandler) waitResponse(ctx context.Context, respCh <-chan *dns.Msg) *dns.Msg {
	var respCount = cap(respCh)
	for {
		select {
		case resp, ok := <-respCh:
			if !ok {
				return nil
			}
			respCount--
			if resp == nil {
				if respCount == 0 {
					return nil
				}
				continue
			}
			if resp.Rcode == dns.RcodeSuccess {
				return resp
			}
			if respCount == 0 {
				return nil
			}

		case <-ctx.Done():
			return nil
		}
	}
}

// NewDNSHandler creates a new health check dns handler
// dnsHealthCheckHandler is expected to be placed at the beginning of the handlers chain
func NewDNSHandler(params Params, opts ...Option) dnsutils.Handler {
	var h = &dnsHealthCheckHandler{
		dnsServerIP: params.DNSServerIP,
		healthHost:  ToWildcardPath(params.HealthHost),
		scheme:      params.Scheme,
		dnsPort:     53,
	}

	for _, o := range opts {
		o(h)
	}
	return h
}

// ToWildcardPath will modify host by adding dot at the end
func ToWildcardPath(host string) string {
	return fmt.Sprintf("%s.", host)
}
