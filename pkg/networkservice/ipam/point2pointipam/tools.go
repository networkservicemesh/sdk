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

	"github.com/RoaringBitmap/roaring"

	"github.com/networkservicemesh/sdk/pkg/tools/cidr"
)

func exclude(prefixes ...string) (*roaring.Bitmap, error) {
	exclude := roaring.New()
	for _, prefix := range prefixes {
		_, ipNet, err := net.ParseCIDR(prefix)
		if err != nil {
			return nil, err
		}
		low := binary.BigEndian.Uint32(cidr.NetworkAddress(ipNet).To4())
		high := binary.BigEndian.Uint32(cidr.BroadcastAddress(ipNet).To4()) + 1

		exclude.AddRange(uint64(low), uint64(high))
	}
	return exclude, nil
}

func addrToInt(addr string) uint32 {
	ip, _, _ := net.ParseCIDR(addr)
	return binary.BigEndian.Uint32(ip.To4())
}
