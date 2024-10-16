// Copyright (c) 2024 Cisco and/or its affiliates.
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

// Package dualstack provides tools for managing both ipv4 and ipv6 addresses
package dualstack

import (
	"net"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

// IPPool holds available IPv4 and IPv6 addresses in the structure of red-black tree
type IPPool struct {
	IPv4IPPool *ippool.IPPool
	IPv6IPPool *ippool.IPPool
}

// New instantiates a dualstack ip pool as red-black tree
func New() *IPPool {
	pool := new(IPPool)
	pool.IPv4IPPool = ippool.New(net.IPv4len)
	pool.IPv6IPPool = ippool.New(net.IPv6len)
	return pool
}

// AddIPNet - adds ip addresses from network to the pool
func (p *IPPool) AddIPNet(ipNet *net.IPNet) {
	if ipNet.IP.To4() != nil {
		p.IPv4IPPool.AddNet(ipNet)
		return
	}
	p.IPv6IPPool.AddNet(ipNet)
}

// AddIPNetString - adds ip addresses from network to the pool by string value
func (p *IPPool) AddIPNetString(ipNetString string) {
	_, ipNet, err := net.ParseCIDR(ipNetString)
	if err != nil {
		return
	}
	p.AddIPNet(ipNet)
}

// ContainsIP checks that pool contains ip
func (p *IPPool) ContainsIP(ip net.IP) bool {
	if ip.To4() != nil {
		return p.IPv4IPPool.Contains(ip)
	}
	return p.IPv6IPPool.Contains(ip)
}

// ContainsIPString parses ip string and checks that pool contains ip
func (p *IPPool) ContainsIPString(in string) bool {
	return p.ContainsIP(net.ParseIP(in))
}

// ContainsIPNet checks that pool contains ip net
func (p *IPPool) ContainsIPNet(ipNet *net.IPNet) bool {
	if ipNet.IP.To4() != nil {
		return p.IPv4IPPool.ContainsNet(ipNet)
	}
	return p.IPv6IPPool.ContainsNet(ipNet)
}

// ContainsIPNetString parses ip net and checks that pool contains ip net
func (p *IPPool) ContainsIPNetString(ipNetRaw string) bool {
	ip, _, _ := net.ParseCIDR(ipNetRaw)
	if ip.To4() != nil {
		return p.IPv4IPPool.ContainsNetString(ipNetRaw)
	}
	return p.IPv6IPPool.ContainsNetString(ipNetRaw)
}

// PullIP - returns requested IP address from the pool
func (p *IPPool) PullIP(ip net.IP) (*net.IPNet, error) {
	if ip.To4() != nil {
		return p.IPv4IPPool.PullIP(ip)
	}
	return p.IPv6IPPool.PullIP(ip)
}

// PullIPString - returns requested IP address from the pool by string
func (p *IPPool) PullIPString(in string) (*net.IPNet, error) {
	ip, _, err := net.ParseCIDR(in)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s as a CIDR", in)
	}
	return p.PullIP(ip)
}
