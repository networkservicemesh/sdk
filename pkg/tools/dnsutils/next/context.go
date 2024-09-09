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

package next

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils"
)

type key struct{}

func withNextHandler(parent context.Context, next dnsutils.Handler) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, key{}, next)
}

// Handler returns next handler or tail.
func Handler(ctx context.Context) dnsutils.Handler {
	rv, ok := ctx.Value(key{}).(dnsutils.Handler)
	if ok && rv != nil {
		return rv
	}
	return &tailHandler{}
}

type tailHandler struct{}

func (*tailHandler) ServeDNS(ctx context.Context, rp dns.ResponseWriter, m *dns.Msg) {}
