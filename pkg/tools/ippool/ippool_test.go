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
	ipPool := New(net.IPv4len)

	ipPool.AddString("192.168.1.255")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddString("192.168.3.0")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddString("192.168.2.0")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddString("192.168.2.255")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_AddRange(t *testing.T) {
	ipPool := New(net.IPv4len)

	ipPool.AddNetString("192.168.1.0/31")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddNetString("192.168.10.0/24")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("192.168.1.0/24")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("192.168.11.0/24")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_Contains(t *testing.T) {
	ipPool := NewWithNetString("192.168.0.0/16")

	require.True(t, ipPool.ContainsString("192.168.0.1"))
	require.False(t, ipPool.ContainsString("192.167.0.1"))
}

func TestIPPoolTool_Exclude(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.0.0.0/8")
	require.NoError(t, err)
	ipPool := NewWithNet(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("192.255.0.0/16")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))

	_, ipNet, err = net.ParseCIDR("192.0.1.0/24")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(2))

	_, ipNet, err = net.ParseCIDR("192.0.0.0/16")
	require.NoError(t, err)
	ipPool.Exclude(ipNet)
	require.Equal(t, ipPool.size, uint64(1))
}

func TestIPPoolTool_Pull(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	ip, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.0")

	ipPool.ExcludeString("192.0.0.0/24")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.1.0")

	ipPool.AddNetString("192.0.0.6/31")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.6")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.0.7")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "192.0.1.1")

	ipPool.ExcludeString("192.0.0.0/8")
	ip, err = ipPool.Pull()
	require.Error(t, err)
}

func TestIPPoolTool_PullP2PAddrs(t *testing.T) {
	ipPool := NewWithNetString("192.0.0.0/8")
	require.NotNil(t, ipPool)

	excludedPool := NewWithNetString("192.0.0.0/32")
	excluded2Pool := NewWithNetString("192.0.0.2/32")
	srcIPNet, dstIPNet, err := ipPool.PullP2PAddrs(excludedPool, excluded2Pool)
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "192.0.0.1/32")
	require.Equal(t, dstIPNet.String(), "192.0.0.3/32")

	srcIPNet, dstIPNet, err = ipPool.PullP2PAddrs()
	require.NoError(t, err)
	require.Equal(t, srcIPNet.String(), "192.0.0.0/32")
	require.Equal(t, dstIPNet.String(), "192.0.0.2/32")

	srcIP, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, srcIP.String(), "192.0.0.4")

}

func TestIPPoolTool_IPv6Add(t *testing.T) {
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

func TestIPPoolTool_IPv6AddRange(t *testing.T) {
	ipPool := New(net.IPv6len)

	ipPool.AddNetString("::1:0/127")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.AddNetString("::10:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("::1:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.AddNetString("::11:0/112")
	require.Equal(t, ipPool.size, uint64(2))
}

func TestIPPoolTool_IPv6Contains(t *testing.T) {
	ipPool := NewWithNetString("::/64")

	require.True(t, ipPool.ContainsString("::0:1"))
	require.False(t, ipPool.ContainsString("0:1::0:1"))
}

func TestIPPoolTool_IPv6Exclude(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.ExcludeString("0:0:ffff:ffff::/64")
	require.Equal(t, ipPool.size, uint64(1))

	ipPool.ExcludeString("::1:0/112")
	require.Equal(t, ipPool.size, uint64(2))

	ipPool.ExcludeString("::/64")
	require.Equal(t, ipPool.size, uint64(1))
}

func TestIPPoolTool_IPv6Pull(t *testing.T) {
	ipPool := NewWithNetString("::/32")
	require.NotNil(t, ipPool)

	ip, err := ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::")

	ipPool.ExcludeString("::/112")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:0")

	ipPool.AddNetString("::6/127")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::6")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::7")
	ip, err = ipPool.Pull()
	require.NoError(t, err)
	require.Equal(t, ip.String(), "::1:1")

	ipPool.ExcludeString("::/32")
	ip, err = ipPool.Pull()
	require.Error(t, err)
}
