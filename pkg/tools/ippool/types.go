// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package ippool

import "math"

type ipAddress struct {
	high, low uint64
}

type ipRange struct {
	start, end *ipAddress
}

func (b *ipAddress) Clone() *ipAddress {
	return &ipAddress{
		high: b.high,
		low:  b.low,
	}
}

func (b *ipAddress) Compare(ip *ipAddress) int {
	switch {
	case ip.high > b.high:
		return 1
	case ip.high < b.high:
		return -1
	case ip.low > b.low:
		return 1
	case ip.low < b.low:
		return -1
	}
	return 0
}

func (b *ipAddress) Equal(ip *ipAddress) bool {
	return b.high == ip.high && b.low == ip.low
}

func (b *ipAddress) IsFirst() bool {
	return b.high == 0 && b.low == 0
}

func (b *ipAddress) IsLast() bool {
	return b.high == math.MaxUint64 && b.low == math.MaxUint64
}

func (b *ipAddress) Prev() *ipAddress {
	r := b.Clone()
	if r.low > 0 {
		r.low--
	} else {
		r.low = math.MaxUint64
		r.high--
	}
	return r
}

func (b *ipAddress) Next() *ipAddress {
	r := b.Clone()
	if r.low < math.MaxUint64 {
		r.low++
	} else {
		r.low = 0
		r.high++
	}
	return r
}

func (b *ipRange) Clone() *ipRange {
	return &ipRange{
		start: b.start.Clone(),
		end:   b.end.Clone(),
	}
}

// Compare - compares with ip address. Returns 0 if crossing, -1 if ipR less than this, 1 otherwise
func (b *ipRange) Compare(ip *ipAddress) int {
	switch {
	case ip.high > b.end.high:
		return 1
	case ip.high < b.start.high:
		return -1
	case ip.high == b.end.high && ip.low > b.end.low:
		return 1
	case ip.high == b.start.high && ip.low < b.start.low:
		return -1
	}
	return 0
}

// CompareRange - compares ip ranges. Returns
//
//	0 if crossing,
//	-1 if ipR less than this and this range continues ipR range,
//	-2 if ipR less than this and this range does not continues ipR range,
//	1 if ipR bigger than this and ipR continues this range,
//	2 if ipR bigger than this and ipR does not continues this range,
func (b *ipRange) CompareRange(ipR *ipRange) int {
	if b.Compare(ipR.start) < 0 && b.Compare(ipR.end) < 0 {
		if !ipR.end.IsLast() && b.Compare(ipR.end.Next()) == 0 {
			return -1
		}
		return -2
	}
	if b.Compare(ipR.start) > 0 && b.Compare(ipR.end) > 0 {
		if !ipR.end.IsFirst() && b.Compare(ipR.start.Prev()) == 0 {
			return 1
		}
		return 2
	}
	return 0
}

// Unite - creates a union range from first available address to last one
func (b *ipRange) Unite(ipR *ipRange) *ipRange {
	r := b.Clone()
	if r.Compare(ipR.start) < 0 {
		r.start = ipR.start.Clone()
	}
	if r.Compare(ipR.end) > 0 {
		r.end = ipR.end.Clone()
	}
	return r
}

// Intersect - creates a combine range including IP addresses from both source ranges at the same time
func (b *ipRange) Intersect(ipR *ipRange) *ipRange {
	if b.CompareRange(ipR) != 0 {
		return nil
	}
	r := b.Clone()
	if r.Compare(ipR.start) > 0 {
		r.start = ipR.start.Clone()
	}
	if r.Compare(ipR.end) < 0 {
		r.end = ipR.end.Clone()
	}
	return r
}

// Sub - creates two ranges including all addresses from this range and excluding ipR
func (b *ipRange) Sub(ipR *ipRange) (leftR, rightR *ipRange) {
	if b.CompareRange(ipR) != 0 {
		return b.Clone(), nil
	}

	if b.start.Compare(ipR.start) > 0 {
		leftR = &ipRange{
			start: b.start.Clone(),
			end:   ipR.start.Prev(),
		}
	}

	if b.end.Compare(ipR.end) < 0 {
		rightR = &ipRange{
			start: ipR.end.Next(),
			end:   b.end.Clone(),
		}
	}
	return
}
