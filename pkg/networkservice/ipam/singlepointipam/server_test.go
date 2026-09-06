// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
// Copyright (c) 2020-2023 Nordix and its affiliates.
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
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/singlepointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func newIpamServer(networks ...*net.IPNet) networkservice.NetworkServiceServer {
	ipamNetworks := []*singlepointipam.IpamNet{}
	for _, n := range networks {
		ipamNetworks = append(ipamNetworks, &singlepointipam.IpamNet{Network: n})
	}
	return newIpamNetServer(ipamNetworks...)
}

func newIpamNetServer(networks ...*singlepointipam.IpamNet) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		metadata.NewServer(),
		singlepointipam.NewServer(networks...),
	)
}

func newIpamNet(ipNet *net.IPNet) *singlepointipam.IpamNet {
	return &singlepointipam.IpamNet{
		Network: ipNet,
	}
}

func newRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}
}

func validateConn(t *testing.T, conn *networkservice.Connection, src string) {
	require.Equal(t, src, conn.Context.IpContext.SrcIpAddrs[0])
}

func validateConns(t *testing.T, conn *networkservice.Connection, srcs []string) {
	require.Equal(t, len(srcs), len(conn.Context.IpContext.SrcIpAddrs))
	for i, src := range srcs {
		require.Equal(t, src, conn.Context.IpContext.SrcIpAddrs[i])
	}
}

//nolint:dupl
func TestServer(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.1/16")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.2/16")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.1/16")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.3/16")
}

//nolint:dupl
func TestServerIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "fe80::1/64")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "fe80::2/64")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "fe80::1/64")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "fe80::3/64")
}

func TestNilPrefixes(t *testing.T) {
	srv := newIpamServer()
	_, err := srv.Request(context.Background(), newRequest())
	require.Error(t, err)

	_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
	require.NoError(t, err)

	srv = newIpamServer(nil, ipNet, nil)
	_, err = srv.Request(context.Background(), newRequest())
	require.Error(t, err)
}

func TestNilPrefixesIPv6(t *testing.T) {
	srv := newIpamServer()
	_, err := srv.Request(context.Background(), newRequest())
	require.Error(t, err)

	_, ipNet, err := net.ParseCIDR("fe80::/128")
	require.NoError(t, err)

	srv = newIpamServer(nil, ipNet, nil)
	_, err = srv.Request(context.Background(), newRequest())
	require.Error(t, err)
}

//nolint:dupl
func TestExclude32Prefix(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.5/32"}
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.1.2/24")

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.5/32"}
	conn2, err := srv.Request(context.Background(), req2)
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.1.4/24")

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.5/32"}
	conn3, err := srv.Request(context.Background(), req3)
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.1.6/24")
}

//nolint:dupl
func TestExclude128PrefixIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::1:0/112")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::1:1/128", "fe80::1:3/128", "fe80::1:5/128"}
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "fe80::1:2/112")

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::1:1/128", "fe80::1:3/128", "fe80::1:5/128"}
	conn2, err := srv.Request(context.Background(), req2)
	require.NoError(t, err)
	validateConn(t, conn2, "fe80::1:4/112")

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::1:1/128", "fe80::1:3/128", "fe80::1:5/128"}
	conn3, err := srv.Request(context.Background(), req3)
	require.NoError(t, err)
	validateConn(t, conn3, "fe80::1:6/112")
}

func TestBroadCastIP(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("10.143.248.96/28")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "10.143.248.97/28")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "10.143.248.98/28")

	for i := 0; i < 11; i++ {
		_, err = srv.Request(context.Background(), newRequest())
		require.NoError(t, err)
	}

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "10.143.248.110/28")

	// Broadcast
	_, err = srv.Request(context.Background(), newRequest())
	require.EqualError(t, err, "IPPool is empty")
}

func TestOutOfIPs(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	_, err = srv.Request(context.Background(), req1)
	require.EqualError(t, err, "IPPool is empty")
}

func TestOutOfIPsIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::1:2/127")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "fe80::1:3/127")

	req2 := newRequest()
	_, err = srv.Request(context.Background(), req2)
	require.EqualError(t, err, "IPPool is empty")
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.0/30")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.2/30"}
	conn1, err := srv.Request(context.Background(), req1)
	require.Nil(t, conn1)
	require.EqualError(t, err, "IPPool is empty")
}

func TestAllIPsExcludedIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::1:0/126")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::1:2/126"}
	conn1, err := srv.Request(context.Background(), req1)
	require.Nil(t, conn1)
	require.EqualError(t, err, "IPPool is empty")
}

//nolint:dupl
func TestRefreshRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.2/16")

	req = newRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.2/16")

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.5/16")
}

//nolint:dupl
func TestRefreshRequestIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::1/128"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "fe80::2/64")

	req = newRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "fe80::2/64")

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"fe80::/126"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "fe80::5/64")
}

func TestNextError(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	_, err = next.NewNetworkServiceServer(srv, injecterror.NewServer()).Request(context.Background(), newRequest())
	require.Error(t, err)

	conn, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.1/16")
}

func TestRefreshNextError(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newRequest()
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.1/16")

	req.Connection = conn.Clone()
	_, err = next.NewNetworkServiceServer(srv, injecterror.NewServer()).Request(context.Background(), newRequest())
	require.Error(t, err)

	conn, err = srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.2/16")
}

//nolint:dupl
func TestServers(t *testing.T) {
	_, ipNet1, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)
	_, ipNet2, err := net.ParseCIDR("fd00::/8")
	require.NoError(t, err)

	srv := chain.NewNetworkServiceServer(
		newIpamServer(ipNet1),
		newIpamServer(ipNet2),
	)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConns(t, conn1, []string{"192.168.0.1/16", "fd00::1/8"})

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConns(t, conn2, []string{"192.168.0.2/16", "fd00::2/8"})

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConns(t, conn3, []string{"192.168.0.1/16", "fd00::1/8"})

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConns(t, conn4, []string{"192.168.0.3/16", "fd00::3/8"})
}

//nolint:dupl
func TestRefreshRequestMultiServer(t *testing.T) {
	_, ipNet1, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)
	_, ipNet2, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := chain.NewNetworkServiceServer(
		newIpamServer(ipNet1),
		newIpamServer(ipNet2),
	)

	req := newRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32", "fe80::1/128"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConns(t, conn, []string{"192.168.0.2/16", "fe80::2/64"})

	req = newRequest()
	req.Connection = conn
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConns(t, conn, []string{"192.168.0.2/16", "fe80::2/64"})

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30", "fe80::1/126"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConns(t, conn, []string{"192.168.0.5/16", "fe80::5/64"})
}

//nolint:dupl
func TestIPv4Range(t *testing.T) {
	sip, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)
	eip := net.ParseIP("192.168.4.1")
	require.NotNil(t, eip)

	ipamNet := newIpamNet(ipNet)
	err = ipamNet.AddRange(sip, eip)
	require.NoError(t, err)
	srv := newIpamNetServer(ipamNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.3.5/16")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.3.6/16")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.3.5/16")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.3.7/16")
}

// TestIPv4RangeServers tests that an IP subnet can be split up between multiple
// indenpendent singlepointipam servers to avoid collision when it comes to IP allocation
//
//nolint:dupl
func TestIPv4RangeServers(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.0.0/24")
	require.NoError(t, err)

	srvs := []networkservice.NetworkServiceServer{
		newIpamNetServer(
			func() *singlepointipam.IpamNet {
				n := newIpamNet(ipNet)
				err = n.AddRange(net.ParseIP("192.168.0.1"), net.ParseIP("192.168.0.2"))
				require.NoError(t, err)
				err = n.AddRange(net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.6"))
				require.NoError(t, err)
				return n
			}(),
		),
		newIpamNetServer(
			func() *singlepointipam.IpamNet {
				n := newIpamNet(ipNet)
				err = n.AddRange(net.ParseIP("192.168.0.3"), net.ParseIP("192.168.0.4"))
				require.NoError(t, err)
				err = n.AddRange(net.ParseIP("192.168.0.7"), net.ParseIP("192.168.0.8"))
				require.NoError(t, err)
				return n
			}(),
		),
	}

	conn, err := srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.2/24")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.4/24")

	conn, err = srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.5/24")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.7/24")

	conn, err = srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.6/24")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.8/24")

	_, err = srvs[0].Request(context.Background(), newRequest())
	require.Error(t, err)

	_, err = srvs[1].Request(context.Background(), newRequest())
	require.Error(t, err)
}

//nolint:dupl
func TestIPv6Range(t *testing.T) {
	sip, ipNet, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)
	eip := net.ParseIP("fe80::ff")
	require.NotNil(t, eip)

	ipamNet := newIpamNet(ipNet)
	err = ipamNet.AddRange(sip, eip)
	require.NoError(t, err)
	srv := newIpamNetServer(ipamNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "fe80::1/64")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "fe80::2/64")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "fe80::1/64")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "fe80::3/64")
}

// TestIPv6RangeServers tests that an IPv6 subnet can be split up between multiple
// indenpendent singlepointipam servers to avoid collision when it comes to IP allocation
//
//nolint:dupl
func TestIPv6RangeServers(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("2dea::1/48")
	require.NoError(t, err)

	srvs := []networkservice.NetworkServiceServer{
		newIpamNetServer(
			func() *singlepointipam.IpamNet {
				n := newIpamNet(ipNet)
				err = n.AddRange(net.ParseIP("2dea::a:0"), net.ParseIP("2dea::a:1"))
				require.NoError(t, err)
				err = n.AddRange(net.ParseIP("2dea::f:1"), net.ParseIP("2dea::f:2"))
				require.NoError(t, err)
				return n
			}(),
		),
		newIpamNetServer(
			func() *singlepointipam.IpamNet {
				n := newIpamNet(ipNet)
				err = n.AddRange(net.ParseIP("2dea::a:2"), net.ParseIP("2dea::a:3"))
				require.NoError(t, err)
				err = n.AddRange(net.ParseIP("2dea::f:ff01"), net.ParseIP("2dea::f:ff02"))
				require.NoError(t, err)
				return n
			}(),
		),
	}

	conn, err := srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::a:1/48")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::a:3/48")

	conn, err = srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::f:1/48")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::f:ff01/48")

	conn, err = srvs[0].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::f:2/48")

	conn, err = srvs[1].Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn, "2dea::f:ff02/48")

	_, err = srvs[0].Request(context.Background(), newRequest())
	require.Error(t, err)

	_, err = srvs[1].Request(context.Background(), newRequest())
	require.Error(t, err)
}
