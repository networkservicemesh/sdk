// Copyright (c) 2023 Nordix and its affiliates.
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

package singlepointipam

import (
	"encoding/binary"
	"net"

	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/tools/ippool"
)

// IpamNet - IP network with optional IP ranges to limit address scope
type IpamNet struct {
	Network *net.IPNet     // the subnetwork represented by a particular IpamNet instance
	Pool    *ippool.IPPool // optional: limit IpamNet's scope by specifying an IP pool within Network
}

// NewIpamNet - creates a new IpamNet
func NewIpamNet() *IpamNet {
	return &IpamNet{}
}

// AddRange - adds range of IPs defined by firstip lastip tuple to customize IpamNet's scope.
func (ipn *IpamNet) AddRange(firstip, lastip net.IP) error {
	if ipn.Network == nil {
		return errors.Errorf("network not set")
	}

	// both must belong to the network
	if !ipn.Network.Contains(firstip) || !ipn.Network.Contains(lastip) {
		return errors.Errorf("%s-%s invalid range for CIDR %s", firstip, lastip, ipn.Network.String())
	}

	// firstip must precede lastip
	fipHigh := binary.BigEndian.Uint64(firstip.To16()[:8])
	lipHigh := binary.BigEndian.Uint64(lastip.To16()[:8])
	if fipHigh > lipHigh ||
		fipHigh == lipHigh && binary.BigEndian.Uint64(firstip.To16()[8:]) > binary.BigEndian.Uint64(lastip.To16()[8:]) {
		return errors.Errorf("%s-%s invalid range start", firstip, lastip)
	}

	if ipn.Pool == nil {
		ipn.Pool = ippool.New(len(ipn.Network.IP))
	}

	ipn.Pool.AddRange(firstip, lastip)
	return nil
}
