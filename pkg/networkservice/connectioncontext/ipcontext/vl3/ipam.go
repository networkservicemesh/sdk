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

package vl3

import (
	"context"
	"net"
	"sync"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type vl3IPAM struct {
	sync.Mutex
	self             net.IPNet
	ipPool           *ippool.IPPool
	excludedPrefixes map[string]struct{}
	clientMask       uint8
}

func (p *vl3IPAM) isInitialized() bool {
	p.Lock()
	defer p.Unlock()

	return p.ipPool != nil
}

func (p *vl3IPAM) selfAddress() *net.IPNet {
	p.Lock()
	defer p.Unlock()
	return &net.IPNet{
		IP: p.self.IP,
		Mask: net.CIDRMask(
			int(p.clientMask),
			int(p.clientMask),
		),
	}
}

func (p *vl3IPAM) selfPrefix() *net.IPNet {
	p.Lock()
	defer p.Unlock()
	r := p.self
	return &r
}
func (p *vl3IPAM) globalIPNet() *net.IPNet {
	p.Lock()
	defer p.Unlock()
	return &net.IPNet{
		IP: p.self.IP,
		Mask: net.CIDRMask(
			int(p.clientMask)/2,
			int(p.clientMask),
		),
	}
}

func (p *vl3IPAM) allocate() (*net.IPNet, error) {
	p.Lock()
	defer p.Unlock()

	ip, err := p.ipPool.Pull()
	if err != nil {
		return nil, err
	}

	r := &net.IPNet{
		IP: ip,
		Mask: net.CIDRMask(
			int(p.clientMask),
			int(p.clientMask),
		),
	}

	p.excludedPrefixes[r.String()] = struct{}{}
	return r, nil
}

func (p *vl3IPAM) freeIfAllocated(ipNet string) {
	p.Lock()
	defer p.Unlock()

	if _, ok := p.excludedPrefixes[ipNet]; ok {
		delete(p.excludedPrefixes, ipNet)
		p.ipPool.AddNetString(ipNet)
	}
}

func (p *vl3IPAM) isExcluded(ipNet string) bool {
	p.Lock()
	defer p.Unlock()

	_, r := p.excludedPrefixes[ipNet]
	return r
}

func (p *vl3IPAM) reset(ctx context.Context, prefix string, excludePrefies []string) {
	p.Lock()
	defer p.Unlock()

	_, ipNet, err := net.ParseCIDR(prefix)
	if err != nil {
		log.FromContext(ctx).Error(err.Error())
		return
	}

	p.self = *ipNet
	p.ipPool = ippool.NewWithNet(ipNet)
	p.excludedPrefixes = make(map[string]struct{})
	p.clientMask = net.IPv6len * 8
	if len(p.self.IP) == net.IPv4len {
		p.clientMask = net.IPv4len * 8
	}
	selfAddress := &net.IPNet{
		IP: p.self.IP,
		Mask: net.CIDRMask(
			int(p.clientMask),
			int(p.clientMask),
		),
	}
	p.excludedPrefixes[selfAddress.String()] = struct{}{}
	p.ipPool.Exclude(selfAddress)

	for _, excludePrefix := range excludePrefies {
		p.ipPool.ExcludeString(excludePrefix)
		p.excludedPrefixes[excludePrefix] = struct{}{}
	}
}
