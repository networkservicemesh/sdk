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

// Package fanout sends incoming queries in parallel to few endpoints
package fanout

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type fanoutHandler struct {
	dnsPort uint16
}

func (h *fanoutHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	connectTO := clienturlctx.ClientURLs(ctx)
	responseCh := make(chan *dns.Msg, len(connectTO))

	deadline, _ := ctx.Deadline()
	timeout := time.Until(deadline)

	if len(connectTO) == 0 {
		log.FromContext(ctx).WithField("fanoutHandler", "ServeDNS").Error("no urls to fanout")
		dns.HandleFailed(rw, msg)
		return
	}

	for i := 0; i < len(connectTO); i++ {
		go func(u *url.URL, msg *dns.Msg) {
			client := dns.Client{
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

			resp, _, err := client.Exchange(msg, address)
			if err != nil {
				log.FromContext(ctx).WithField("fanoutHandler", "ServeDNS").Warnf("got an error during exchanging with address %v: %v", address, err.Error())
				responseCh <- nil
				return
			}

			responseCh <- resp
		}(&connectTO[i], msg.Copy())
	}

	resp := h.waitResponse(ctx, responseCh)

	if resp == nil {
		dns.HandleFailed(rw, msg)
		return
	}

	if err := rw.WriteMsg(resp); err != nil {
		log.FromContext(ctx).WithField("fanoutHandler", "ServeDNS").Warnf("got an error during write the message: %v", err.Error())
		dns.HandleFailed(rw, msg)
		return
	}

	next.Handler(ctx).ServeDNS(ctx, rw, msg)
}

func (h *fanoutHandler) waitResponse(ctx context.Context, respCh <-chan *dns.Msg) *dns.Msg {
	respCount := cap(respCh)
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

// NewDNSHandler creates a new dns handler instance that sends incoming queries in parallel to few endpoints.
func NewDNSHandler(opts ...Option) dnsutils.Handler {
	h := &fanoutHandler{
		dnsPort: 53,
	}
	for _, o := range opts {
		o(h)
	}
	return h
}
