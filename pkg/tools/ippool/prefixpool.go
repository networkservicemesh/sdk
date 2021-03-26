// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package ippool

import "net"

// PrefixPool - keeps prefixes for both IPv4 and IPv6 addresses
type PrefixPool struct {
	ip4, ip6 *IPPool
}

// NewPool - Creates new PrefixPool with initial prefixes list
func NewPool(prefixes ...string) (*PrefixPool, error) {
	pool := &PrefixPool{
		ip4: New(net.IPv4len),
		ip6: New(net.IPv6len),
	}

	err := pool.AddPrefixes(prefixes...)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

// AddPrefixes - adds prefixes to the pool
func (pool *PrefixPool) AddPrefixes(prefixes ...string) error {
	for _, prefix := range prefixes {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			return err
		}
		pool.ip4.AddNet(ipNet)
		pool.ip6.AddNet(ipNet)
	}
	return nil
}

// ExcludePrefixes - removes prefixes from the pool
func (pool *PrefixPool) ExcludePrefixes(prefixes ...string) error {
	for _, prefix := range prefixes {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			return err
		}
		pool.ip4.Exclude(ipNet)
		pool.ip6.Exclude(ipNet)
	}
	return nil
}

// GetPrefixes - returns the list of saved prefixes
func (pool *PrefixPool) GetPrefixes() []string {
	return append(pool.ip4.GetPrefixes(), pool.ip6.GetPrefixes()...)
}
