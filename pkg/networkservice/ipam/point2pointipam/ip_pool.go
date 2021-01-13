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

package point2pointipam

import (
	"encoding/binary"
	"net"
	"sync"

	"github.com/RoaringBitmap/roaring"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

type ipPool struct {
	freeIPs *roaring.Bitmap
	lock    sync.Mutex
}

func newIPPool(ipNet *net.IPNet) *ipPool {
	p := &ipPool{
		freeIPs: roaring.New(),
	}

	networkInt := binary.BigEndian.Uint32(cidr.NetworkAddress(ipNet))
	broadcastInt := binary.BigEndian.Uint32(cidr.BroadcastAddress(ipNet))

	p.freeIPs.AddRange(uint64(networkInt), uint64(broadcastInt+1))

	return p
}

func (p *ipPool) getP2PAddrs(exclude *roaring.Bitmap) (dstAddr, srcAddr string, err error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	available := p.freeIPs.Clone()
	if available.IsEmpty() {
		return "", "", errors.New("IP pool is empty")
	}

	available = roaring.AndNot(available, exclude)
	if available.IsEmpty() {
		return "", "", errors.New("all free IP addresses are excluded")
	}
	dstInt := available.Minimum()

	available.Remove(dstInt)
	if available.IsEmpty() {
		return "", "", errors.New("all free IP addresses are excluded")
	}
	srcInt := available.Minimum()

	p.freeIPs.Remove(dstInt)
	p.freeIPs.Remove(srcInt)

	dstIP := &net.IPNet{
		IP:   make(net.IP, 4),
		Mask: p2pMask(),
	}
	binary.BigEndian.PutUint32(dstIP.IP, dstInt)

	srcIP := &net.IPNet{
		IP:   make(net.IP, 4),
		Mask: p2pMask(),
	}
	binary.BigEndian.PutUint32(srcIP.IP, srcInt)

	return dstIP.String(), srcIP.String(), nil
}

func p2pMask() net.IPMask {
	return net.CIDRMask(32, 32) // x.x.x.x/32
}

func (p *ipPool) freeAddrs(addrs ...string) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, addr := range addrs {
		p.freeIPs.Add(addrToInt(addr))
	}
}
