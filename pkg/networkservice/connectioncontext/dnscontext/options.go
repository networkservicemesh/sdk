// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
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

package dnscontext

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/dnsconfig"
)

// DNSOption is applying options for DNS client.
type DNSOption interface {
	apply(*dnsContextClient)
}

type applyFunc func(*dnsContextClient)

func (f applyFunc) apply(c *dnsContextClient) {
	f(c)
}

// WithResolveConfigPath sets resolv.conf path for resonvConfigHandler.
func WithResolveConfigPath(path string) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.resolveConfigPath = path
	})
}

// WithDefaultNameServerIP sets default nameserver ip for resolvConfigHandler.
func WithDefaultNameServerIP(ip string) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.defaultNameServerIP = ip
	})
}

// WithDNSConfigsMap sets configs map for DNS client.
func WithDNSConfigsMap(configsMap *dnsconfig.Map) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.dnsConfigsMap = configsMap
	})
}

// WithChainContext sets chain context for DNS client.
func WithChainContext(ctx context.Context) DNSOption {
	return applyFunc(func(c *dnsContextClient) {
		c.chainContext = ctx
	})
}
