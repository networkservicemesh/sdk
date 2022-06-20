// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package dnscontext

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsutils/dnsconfigs"
)

// DNSOption is applying options for DNS client.
type DNSOption interface {
	apply(*dnsContextClient)
}

type applyFunc func(*dnsContextClient)

func (f applyFunc) apply(c *dnsContextClient) {
	f(c)
}

// WithDNSIPsMap sets specific corefile path for DNS client.
func WithDNSIPsMap(ipsMap *dnsconfigs.DNSServerIpMap) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.dnsIPsMap = ipsMap
	})
}

// WithSearchDomainsMap sets specific resolve config file path for DNS client.
func WithSearchDomainsMap(domainsMap *dnsconfigs.SearchDomainsMap) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.searchDomainsMap = domainsMap
	})
}

// WithChainContext sets chain context for DNS server.
func WithChainContext(ctx context.Context) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.chainContext = ctx
	})
}
