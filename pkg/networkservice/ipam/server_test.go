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

package ipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam"
)

func newP2PRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					DstIpRequired: true,
				},
			},
		},
	}
}

func newIPNetRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					DstIpRequired: false,
				},
			},
		},
	}
}

func newIpamServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		ipam.NewServer(prefixes...),
	)
}

func TestServer(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newP2PRequest())
	require.NoError(t, err)

	require.Equal(t, "192.168.0.0/31", conn1.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.1/31", conn1.Context.IpContext.SrcIpAddr)

	conn2, err := srv.Request(context.Background(), newP2PRequest())
	require.NoError(t, err)

	require.Equal(t, "192.168.0.2/31", conn2.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.3/31", conn2.Context.IpContext.SrcIpAddr)

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newP2PRequest())
	require.NoError(t, err)

	require.Equal(t, "192.168.0.0/31", conn3.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.1/31", conn3.Context.IpContext.SrcIpAddr)

	conn4, err := srv.Request(context.Background(), newP2PRequest())
	require.NoError(t, err)

	require.Equal(t, "192.168.0.4/31", conn4.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.5/31", conn4.Context.IpContext.SrcIpAddr)
}

func TestNilPrefixes(t *testing.T) {
	srv := newIpamServer()
	_, err := srv.Request(context.Background(), newP2PRequest())
	require.Error(t, err)

	_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
	require.NoError(t, err)

	srv = newIpamServer(nil, ipNet, nil)
	_, err = srv.Request(context.Background(), newP2PRequest())
	require.Error(t, err)
}

func TestExclude32Prefix(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	// Test center of assigned
	req1 := newP2PRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	require.Equal(t, "192.168.1.4/31", conn1.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.1.5/31", conn1.Context.IpContext.SrcIpAddr)

	// Test exclude before assigned
	req2 := newP2PRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn2, err := srv.Request(context.Background(), req2)
	require.NoError(t, err)
	require.Equal(t, "192.168.1.6/31", conn2.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.1.7/31", conn2.Context.IpContext.SrcIpAddr)

	// Test after assigned
	req3 := newP2PRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.8/32"}
	conn3, err := srv.Request(context.Background(), req3)
	require.NoError(t, err)
	require.Equal(t, "192.168.1.10/31", conn3.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.1.11/31", conn3.Context.IpContext.SrcIpAddr)
}

func TestOutOfIPs(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newP2PRequest()
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	require.Equal(t, "192.168.1.2/31", conn1.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.1.3/31", conn1.Context.IpContext.SrcIpAddr)

	req2 := newP2PRequest()
	_, err = srv.Request(context.Background(), req2)
	require.Error(t, err)
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newP2PRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.2/31"}
	conn1, err := srv.Request(context.Background(), req1)
	require.Nil(t, conn1)
	require.Error(t, err)
}

func TestIPNetRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn, err := srv.Request(context.Background(), newIPNetRequest())
	require.NoError(t, err)
	require.Equal(t, "192.168.0.1/16", conn.Context.IpContext.SrcIpAddr)

	conn, err = srv.Request(context.Background(), newIPNetRequest())
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/16", conn.Context.IpContext.SrcIpAddr)

	conn, err = srv.Request(context.Background(), newIPNetRequest())
	require.NoError(t, err)
	require.Equal(t, "192.168.0.3/16", conn.Context.IpContext.SrcIpAddr)
}

func TestRefreshIPNetRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newIPNetRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/16", conn.Context.IpContext.SrcIpAddr)

	req = newIPNetRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/16", conn.Context.IpContext.SrcIpAddr)

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.4/16", conn.Context.IpContext.SrcIpAddr)
}

func TestRefreshP2PRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newP2PRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/31", conn.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.3/31", conn.Context.IpContext.SrcIpAddr)

	req = newP2PRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/31", conn.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.3/31", conn.Context.IpContext.SrcIpAddr)

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.4/31", conn.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.5/31", conn.Context.IpContext.SrcIpAddr)
}

func TestChangeRequestCase(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newP2PRequest()
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.0/31", conn.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.1/31", conn.Context.IpContext.SrcIpAddr)

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.DstIpRequired = false
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.1/16", conn.Context.IpContext.SrcIpAddr)

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.DstIpRequired = true
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, "192.168.0.2/31", conn.Context.IpContext.DstIpAddr)
	require.Equal(t, "192.168.0.3/31", conn.Context.IpContext.SrcIpAddr)
}
