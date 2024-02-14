// Copyright (c) 2023-2024 Cisco and/or its affiliates.
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

package sandbox

import (
	"context"
	"testing"
)

// InterdomainBuilder helps configure multi domain setups for unit testing
type InterdomainBuilder struct {
	domains   []*Domain
	ctx       context.Context
	t         *testing.T
	buildFn   func()
	dnsServer *DNSServerEntry
}

// NewInterdomainBuilder creates a new instance of InterdomainBuilder
func NewInterdomainBuilder(ctx context.Context, t *testing.T) *InterdomainBuilder {
	var res = &InterdomainBuilder{
		t:   t,
		ctx: ctx,
	}

	res.buildFn = func() {
		res.dnsServer = NewBuilder(ctx, t).SetNodesCount(0).SetRegistrySupplier(nil).SetupDefaultDNSServer().Build().DNSServer
	}

	return res
}

// BuildDomain configures a new domain
func (b *InterdomainBuilder) BuildDomain(creator func(*Builder) *Builder) *InterdomainBuilder {
	var builder = NewBuilder(b.ctx, b.t).EnableInterdomain()
	creator(builder)
	var fn = b.buildFn
	b.buildFn = func() {
		fn()
		builder.dnsServer = b.dnsServer
		b.domains = append(b.domains, builder.Build())
	}

	return b
}

// Build starts all domains
func (b *InterdomainBuilder) Build() []*Domain {
	b.buildFn()
	b.buildFn = func() {}
	var res = b.domains
	b.domains = nil
	return res
}
