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
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

func TestIPPoolTool_Add(t *testing.T) {
	ipPool := New(net.IPv6len)

	ipPool.Add(net.ParseIP("::1:ffff"))
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.Add(net.ParseIP("::1:0"))
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.Add(net.ParseIP("::2:0"))
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.Add(net.ParseIP("::0:ffff"))
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_AddRange(t *testing.T) {
	ipPool := New(net.IPv6len)

	_, ipNet, err := net.ParseCIDR("::1:0/127")
	require.NoError(t, err)
	ipPool.AddNet(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("::10:0/112")
	require.NoError(t, err)
	ipPool.AddNet(ipNet)
	require.Equal(t, ipPool.size, uint64(2))

	_, ipNet, err = net.ParseCIDR("::1:0/112")
	require.NoError(t, err)
	ipPool.AddNet(ipNet)
	require.Equal(t, ipPool.size, uint64(2))

	_, ipNet, err = net.ParseCIDR("::11:0/112")
	require.NoError(t, err)
	ipPool.AddNet(ipNet)
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_Exclude(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("::/32")
	require.NoError(t, err)
	ipPool := NewWithNet(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("0:0:ffff:ffff::/64")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("::1:0/112")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(2))

	_, ipNet, err = net.ParseCIDR("::/64")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))
}

func TestIPPoolTool_Pull(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("::/32")
	require.NoError(t, err)
	ipPool := NewWithNet(ipNet)

	ip, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::")

	_, ipNet, err = net.ParseCIDR("::/112")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:0")

	_, ipNet, err = net.ParseCIDR("::6/127")
	require.NoError(t, err)
	ipPool.AddNet(ipNet)
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::6")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::7")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:1")

	_, ipNet, err = net.ParseCIDR("::/32")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	ip, err = ipPool.Pull()
	require.Error(t, err)
}
