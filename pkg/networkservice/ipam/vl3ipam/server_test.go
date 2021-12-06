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

package vl3ipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/vl3ipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func newIpamServer(own *net.IPNet, prefixes ...*net.IPNet) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		metadata.NewServer(),
		vl3ipam.NewServer(own, prefixes...),
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

func validateConn(t *testing.T, conn *networkservice.Connection, dst, src, srcRoute string) {
	require.Equal(t, conn.Context.IpContext.DstIpAddrs[0], dst)
	require.Equal(t, conn.Context.IpContext.DstRoutes, []*networkservice.Route{
		{
			Prefix: src,
		},
	})

	require.Equal(t, conn.Context.IpContext.SrcIpAddrs[0], src)
	require.Equal(t, conn.Context.IpContext.SrcRoutes, []*networkservice.Route{
		{
			Prefix: srcRoute,
		},
	})
}

//nolint:dupl
func TestServer(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(nil, ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/32", "192.168.0.1/32", "192.168.0.0/16")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.0/32", "192.168.0.2/32", "192.168.0.0/16")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.0/32", "192.168.0.1/32", "192.168.0.0/16")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.0/32", "192.168.0.3/32", "192.168.0.0/16")
}

//nolint:dupl
func TestServerSetOwn(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	_, own, err := net.ParseCIDR("192.168.0.1/32")
	require.NoError(t, err)

	srv := newIpamServer(own, ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.1/32", "192.168.0.0/32", "192.168.0.0/16")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.1/32", "192.168.0.2/32", "192.168.0.0/16")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.1/32", "192.168.0.0/32", "192.168.0.0/16")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.1/32", "192.168.0.3/32", "192.168.0.0/16")
}

//nolint:dupl
func TestServerIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := newIpamServer(nil, ipNet)

	conn1, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn1, "fe80::/128", "fe80::1/128", "fe80::/64")

	conn2, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "fe80::/128", "fe80::2/128", "fe80::/64")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "fe80::/128", "fe80::1/128", "fe80::/64")

	conn4, err := srv.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "fe80::/128", "fe80::3/128", "fe80::/64")
}

func TestNilPrefixes(t *testing.T) {
	srv := newIpamServer(nil)
	_, err := srv.Request(context.Background(), newRequest())
	require.Error(t, err)

	_, ipNet, err := net.ParseCIDR("192.168.0.1/32")
	require.NoError(t, err)

	srv = newIpamServer(nil, nil, ipNet, nil)
	_, err = srv.Request(context.Background(), newRequest())
	require.Error(t, err)
}

//nolint:dupl
func TestServerExtraPrefix(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(nil, ipNet)

	request := newRequest()
	request.Connection.Context.IpContext.ExtraPrefixes = append(request.Connection.Context.IpContext.ExtraPrefixes, "192.169.0.0/16")
	conn1, err := srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/32", "192.169.0.0/32", "192.168.0.0/16")

	requestNew := newRequest()
	requestNew.Connection.Context.IpContext.ExtraPrefixes = append(requestNew.Connection.Context.IpContext.ExtraPrefixes, "192.169.0.0/16")
	requestNew.Connection.Context.IpContext.SrcIpAddrs = request.Connection.Context.IpContext.SrcIpAddrs
	requestNew.Connection.Context.IpContext.DstRoutes = request.Connection.Context.IpContext.DstRoutes
	conn2, err := srv.Request(context.Background(), requestNew)
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.0/32", "192.169.0.0/32", "192.168.0.0/16")
}

func TestServerDifferentNs(t *testing.T) {
	ns1Name := "ns-1"
	ns2Name := "ns-2"

	_, ipNet, err := net.ParseCIDR("192.168.3.4/16")
	require.NoError(t, err)

	srv := newIpamServer(nil, ipNet)

	request := newRequest()
	request.Connection.NetworkService = ns1Name
	conn1, err := srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/32", "192.168.0.1/32", "192.168.0.0/16")

	request = newRequest()
	request.Connection.NetworkService = ns2Name
	conn2, err := srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.0.2/32", "192.168.0.3/32", "192.168.0.0/16")

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	request = newRequest()
	request.Connection.NetworkService = ns1Name
	conn3, err := srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.0/32", "192.168.0.1/32", "192.168.0.0/16")

	request = newRequest()
	request.Connection.NetworkService = ns1Name
	conn4, err := srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.0.0/32", "192.168.0.4/32", "192.168.0.0/16")
}
