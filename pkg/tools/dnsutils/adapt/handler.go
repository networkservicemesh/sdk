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

// Package adapt provides possible to adapt dns.Handler to dnsutils.Handler
package adapt

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/next"
)

// NewDNSHandler adapts  dns.Handler to dnsutils.Handler.
// handler is candidate for adaption.
func NewDNSHandler(handler dns.Handler) dnsutils.Handler {
	return &adaptedHandler{original: handler}
}

type adaptedHandler struct {
	original dns.Handler
}

func (h *adaptedHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {
	h.original.ServeDNS(rp, m)
	next.Handler(ctx).ServeDNS(ctx, rp, m)
}
