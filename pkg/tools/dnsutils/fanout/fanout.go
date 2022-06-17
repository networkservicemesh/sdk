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
	"net/url"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type fanoutHandler struct {
	getAddressesFn GetAddressesFn
}

func (f *fanoutHandler) ServeDNS(ctx context.Context, rw dns.ResponseWriter, msg *dns.Msg) {
	var connectTO = f.getAddressesFn()
	var responseCh = make(chan *dns.Msg, len(connectTO))

	if len(connectTO) == 0 {
		log.FromContext(ctx).Error("no urls to fanout")
		dns.HandleFailed(rw, msg)
		return
	}

	for i := 0; i < len(connectTO); i++ {
		go func(u *url.URL, msg *dns.Msg) {
			var client = dns.Client{
				Net: u.Scheme,
			}

			var resp, _, err = client.Exchange(msg, "8.8.8.8:53")

			if err != nil {
				log.FromContext(ctx).Warnf("got an error during exchanging: %v", err.Error())
				responseCh <- nil
				return
			}

			responseCh <- resp
		}(&connectTO[i], msg.Copy())
	}

	var resp = f.waitResponse(ctx, responseCh)

	if resp == nil {
		dns.HandleFailed(rw, msg)
		return
	}

	if err := rw.WriteMsg(resp); err != nil {
		log.FromContext(ctx).Warnf("got an error during write the message: %v", err.Error())
		dns.HandleFailed(rw, msg)
		return
	}

	next.Handler(ctx).ServeDNS(ctx, rw, resp)
}

func (f *fanoutHandler) waitResponse(ctx context.Context, respCh <-chan *dns.Msg) *dns.Msg {
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

// WithStaticAddresses sets endpoints as static addresses
func WithStaticAddresses(addresses ...url.URL) GetAddressesFn {
	if len(addresses) == 0 {
		panic("zero addresses are not supported")
	}
	return func() []url.URL {
		return addresses
	}
}

// NewDNSHandler creates a new dns handler instance that sends incoming queries in parallel to few endpoints
// getAddressesFn gets endpoints for fanout
func NewDNSHandler(getAddressesFn GetAddressesFn) dnsutils.Handler {
	if getAddressesFn == nil {
		panic("no addresses provided")
	}
	return &fanoutHandler{getAddressesFn: getAddressesFn}
}

// GetAddressesFn is alias for urls getter function
type GetAddressesFn = func() []url.URL
