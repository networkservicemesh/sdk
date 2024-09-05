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

// Package connect simply connects to the concrete endpoint
package connect

import (
	"context"
	"net/url"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

// NewDNSHandler creates a new dnshandler that simply connects to the endpoint by passed url
// connectTO is endpoint url.
func NewDNSHandler(connectTO *url.URL) dnsutils.Handler {
	return &connectDNSHandler{connectTO: connectTO}
}

type connectDNSHandler struct{ connectTO *url.URL }

func (c *connectDNSHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, msg *dns.Msg) {
	client := dns.Client{
		Net: c.connectTO.Scheme,
	}

	resp, _, err := client.Exchange(msg, c.connectTO.Host)
	if err != nil {
		log.FromContext(ctx).WithField("connectDNSHandler", "ServeDNS").Warnf("got an error during exchanging: %v", err.Error())
		dns.HandleFailed(rp, msg)
		return
	}

	if err = rp.WriteMsg(resp); err != nil {
		log.FromContext(ctx).WithField("connectDNSHandler", "ServeDNS").Warnf("got an error during write the message: %v", err.Error())
		dns.HandleFailed(rp, msg)
		return
	}

	next.Handler(ctx).ServeDNS(ctx, rp, resp)
}
