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

package singlepointipam_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/singlepointipam"
)

func TestRangeIPv4(t *testing.T) {
	ipamNet := singlepointipam.NewIpamNet()
	require.NotNil(t, ipamNet)
	require.Nil(t, ipamNet.Network)
	require.Nil(t, ipamNet.Pool)

	_, ipNet1, err := net.ParseCIDR("192.2.3.4/16")
	require.NoError(t, err)

	firstip := net.ParseIP("192.2.3.4")
	require.NotNil(t, firstip)
	lastip := net.ParseIP("192.2.3.5")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)
	require.Nil(t, ipamNet.Pool)

	ipamNet.Network = ipNet1
	err = ipamNet.AddRange(firstip, lastip)
	require.NoError(t, err)
	require.NotNil(t, ipamNet.Pool)

	firstip = net.ParseIP("192.2.0.155")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("192.2.128.4")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.NoError(t, err)
}

func TestRangeIPv6(t *testing.T) {
	ipamNet := singlepointipam.NewIpamNet()
	require.NotNil(t, ipamNet)
	require.Nil(t, ipamNet.Network)
	require.Nil(t, ipamNet.Pool)

	_, ipNet1, err := net.ParseCIDR("100:100::1/64")
	require.NoError(t, err)

	firstip := net.ParseIP("100:100::1")
	require.NotNil(t, firstip)
	lastip := net.ParseIP("100:100::1")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)
	require.Nil(t, ipamNet.Pool)

	ipamNet.Network = ipNet1
	err = ipamNet.AddRange(firstip, lastip)
	require.NoError(t, err)
	require.NotNil(t, ipamNet.Pool)

	firstip = net.ParseIP("100:100::d:1")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("100:100::f:2")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.NoError(t, err)

	firstip = net.ParseIP("100:100:0:0::ff00")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("100:100:0:0:1::1")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.NoError(t, err)
}

func TestInvalidRangeIPv4(t *testing.T) {
	ipamNet := singlepointipam.NewIpamNet()
	require.NotNil(t, ipamNet)

	_, ipNet1, err := net.ParseCIDR("192.2.0.0/16")
	require.NoError(t, err)
	ipamNet.Network = ipNet1

	firstip := net.ParseIP("192.1.3.4")
	require.NotNil(t, firstip)
	lastip := net.ParseIP("192.2.0.5")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("192.2.0.4")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("200.2.3.4")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("1.2.3.4")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("192.2.0.10")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("192.2.0.100")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("192.2.0.10")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("100::1")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("100::2")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	require.Nil(t, ipamNet.Pool)
}

func TestInvalidRangeIPv6(t *testing.T) {
	ipamNet := &singlepointipam.IpamNet{}
	require.NotNil(t, ipamNet)

	_, ipNet1, err := net.ParseCIDR("a:b:c:d::/64")
	require.NoError(t, err)
	ipamNet.Network = ipNet1

	firstip := net.ParseIP("a:b:c:e::4")
	require.NotNil(t, firstip)
	lastip := net.ParseIP("a:b:c:d::5")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("a:b:c:d::4")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("a:e:c:d::4")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("1:2:3:4::4")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("a:b:c:d::10")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("a:b:c:d::100")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("a:b:c:d::10")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	firstip = net.ParseIP("192.168.100.1")
	require.NotNil(t, firstip)
	lastip = net.ParseIP("192.168.100.2")
	require.NotNil(t, lastip)

	err = ipamNet.AddRange(firstip, lastip)
	require.Error(t, err)

	require.Nil(t, ipamNet.Pool)
}
