// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package point2pointipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func newIpamServer(prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		metadata.NewServer(),
		point2pointipam.NewServer(prefixes...),
	)
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

func validateConn(t *testing.T, conn *networkservice.Connection, dst, src string) {
	require.Equal(t, conn.Context.IpContext.DstIpAddr, dst)
	require.Equal(t, conn.Context.IpContext.DstRoutes, []*networkservice.Route{
		{
			Prefix: src,
		},
	})

	require.Equal(t, conn.Context.IpContext.SrcIpAddr, src)
	require.Equal(t, conn.Context.IpContext.SrcRoutes, []*networkservice.Route{
		{
			Prefix: dst,
		},
	})
}

func TestServer(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/32", "192.168.0.1/32")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.2/32", "192.168.0.3/32")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.0/32", "192.168.0.1/32")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.4/32", "192.168.0.5/32")
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

func TestExclude32Prefix(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.0/24")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	// Test center of assigned
	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.1.0/32", "192.168.1.2/32")

	// Test exclude before assigned
	req2 := newRequest()
	req2.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn2, err := srv.Request(context.Background(), req2)
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.1.4/32", "192.168.1.5/32")

	// Test after assigned
	req3 := newRequest()
	req3.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.1/32", "192.168.1.3/32", "192.168.1.6/32"}
	conn3, err := srv.Request(context.Background(), req3)
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.1.7/32", "192.168.1.8/32")
}

func TestOutOfIPs(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	conn1, err := srv.Request(context.Background(), req1)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.1.2/32", "192.168.1.3/32")

	req2 := newRequest()
	_, err = srv.Request(context.Background(), req2)
	require.Error(t, err)
}

func TestAllIPsExcluded(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req1 := newRequest()
	req1.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.1.2/31"}
	conn1, err := srv.Request(context.Background(), req1)
	require.Nil(t, conn1)
	require.Error(t, err)
}

func TestRefreshRequest(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(ipNet)

	req := newRequest()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/32"}
	conn, err := srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.0/32", "192.168.0.2/32")

	req = newRequest()
	req.Connection.Id = conn.Id
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.0/32", "192.168.0.2/32")

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"192.168.0.1/30"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	validateConn(t, conn, "192.168.0.4/32", "192.168.0.5/32")
}
