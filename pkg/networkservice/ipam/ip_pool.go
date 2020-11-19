// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package ipam

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

var (
	p2pMask net.IPMask = []byte{0xFF, 0xFF, 0xFF, 0xFE}
)

type ipPool struct {
	mask         net.IPMask
	networkInt   uint32
	broadcastInt uint32
	freeIPs      *roaring.Bitmap
	lock         sync.Mutex
}

func newIPPool(ipNet *net.IPNet) *ipPool {
	p := &ipPool{
		mask:         ipNet.Mask,
		networkInt:   binary.BigEndian.Uint32(cidr.NetworkAddress(ipNet)),
		broadcastInt: binary.BigEndian.Uint32(cidr.BroadcastAddress(ipNet)),
		freeIPs:      roaring.New(),
	}

	p.freeIPs.AddRange(uint64(p.networkInt), uint64(p.broadcastInt+1))

	return p
}

func (p *ipPool) getP2PAddrs(exclude *roaring.Bitmap) (dstAddr, srcAddr string, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	available := p.freeIPs.Clone()
	if available.IsEmpty() {
		return "", "", errors.New("IP pool is empty")
	}

	var dstInt, srcInt uint32
	for available = roaring.AndNot(available, exclude); !available.IsEmpty(); {
		dstInt = available.Minimum()
		available.Remove(dstInt)

		srcInt = dstInt | 1
		if available.Contains(srcInt) {
			break
		}
	}
	if available.IsEmpty() {
		return "", "", errors.New("no available IP address found")
	}

	p.freeIPs.Remove(dstInt)
	p.freeIPs.Remove(srcInt)

	dstIPNet := &net.IPNet{
		IP:   make(net.IP, 4),
		Mask: p2pMask,
	}
	binary.BigEndian.PutUint32(dstIPNet.IP, dstInt)

	srcIPNet := &net.IPNet{
		IP:   make(net.IP, 4),
		Mask: p2pMask,
	}
	binary.BigEndian.PutUint32(srcIPNet.IP, srcInt)

	return dstIPNet.String(), srcIPNet.String(), nil
}

func (p *ipPool) getIPNetAddr(exclude *roaring.Bitmap) (addr string, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	available := p.freeIPs.Clone()
	available.Remove(p.networkInt)
	available.Remove(p.broadcastInt)
	if available.IsEmpty() {
		return "", errors.New("IP pool is empty")
	}

	available = roaring.AndNot(available, exclude)
	if available.IsEmpty() {
		return "", errors.New("no available IP address found")
	}

	addrInt := available.Minimum()
	p.freeIPs.Remove(addrInt)

	ipNet := &net.IPNet{
		IP:   make(net.IP, 4),
		Mask: p.mask,
	}
	binary.BigEndian.PutUint32(ipNet.IP, addrInt)

	return ipNet.String(), nil
}

func (p *ipPool) freeAddrs(addrs ...string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, addr := range addrs {
		p.freeIPs.Add(addrToInt(addr))
	}
}
