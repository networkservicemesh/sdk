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

import (
	"encoding/binary"
	"net"
)

func ipRangeFromIPNet(ipNet *net.IPNet) *ipRange {
	startIp := getStartIp(ipNet)
	endIp := getEndIp(ipNet)

	return &ipRange{
		start: ipAddressFromIP(startIp),
		end:   ipAddressFromIP(endIp),
	}
}

func ipAddressFromIP(ip net.IP) *ipAddress {
	if len(ip) == net.IPv4len {
		return &ipAddress{
			high: 0,
			low:  binary.BigEndian.Uint64(ip.To4()),
		}
	}
	return &ipAddress{
		high: binary.BigEndian.Uint64(ip.To16()[:8]),
		low:  binary.BigEndian.Uint64(ip.To16()[8:]),
	}
}

func ipFromIPAddress(addr *ipAddress, size int) net.IP {
	ip := make(net.IP, size)
	if size == net.IPv4len {
		binary.BigEndian.PutUint32(ip, uint32(addr.low))
	} else {
		binary.BigEndian.PutUint64(ip, addr.high)
		binary.BigEndian.PutUint64(ip[8:], addr.low)
	}
	return ip
}

func getStartIp(ipNet *net.IPNet) net.IP {
	return ipNet.IP.Mask(ipNet.Mask)
}

func getEndIp(ipNet *net.IPNet) net.IP {
	out := make(net.IP, len(ipNet.IP))
	for i := 0; i < len(ipNet.IP); i++ {
		out[i] = ipNet.IP[i] | ^ipNet.Mask[i]
	}

	return out
}
