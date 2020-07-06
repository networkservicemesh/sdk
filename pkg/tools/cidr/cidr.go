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

// Package cidr provides common functions useful when working with Classless Inter-Domain Routing (CIDR)
package cidr

import (
	"encoding/binary"
	"net"
)

// NetworkAddress teturns the first IP address of an IP network
func NetworkAddress(ipNet *net.IPNet) net.IP {
	prefixNetwork := ipNet.IP.Mask(ipNet.Mask)
	return prefixNetwork
}

// BroadcastAddress returns the last IP address of an IP network
func BroadcastAddress(ipNet *net.IPNet) net.IP {
	first := NetworkAddress(ipNet)
	ones, bits := ipNet.Mask.Size()
	var shift uint32 = 1
	shift <<= bits - ones
	intip := binary.BigEndian.Uint32(first.To4())
	intip = intip + shift - 1
	last := make(net.IP, 4)
	binary.BigEndian.PutUint32(last, intip)
	return last
}
