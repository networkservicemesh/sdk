// Copyright (c) 2023 Cisco and/or its affiliates.
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

package groupipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/groupipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/singlepointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_NewServer_ShouldFailIfOptionIsNil(t *testing.T) {
	require.Panics(t, func() {
		groupipam.NewServer([][]*net.IPNet{}, groupipam.WithCustomIPAMServer(nil))
	})
}

func Test_NewServer_UsesPoint2pointIPAMByDefault(t *testing.T) {
	requireConns := func(t *testing.T, conn *networkservice.Connection, dsts, srcs []string) {
		for i, dst := range dsts {
			require.Equal(t, conn.GetContext().GetIpContext().GetDstIpAddrs()[i], dst)
			require.Equal(t, conn.GetContext().GetIpContext().GetSrcRoutes()[i].GetPrefix(), dst)
		}
		for i, src := range srcs {
			require.Equal(t, conn.GetContext().GetIpContext().GetSrcIpAddrs()[i], src)
			require.Equal(t, conn.GetContext().GetIpContext().GetDstRoutes()[i].GetPrefix(), src)
		}
	}

	_, ipNet1, err := net.ParseCIDR("172.92.3.4/16")
	require.NoError(t, err)
	_, ipNet2, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := groupipam.NewServer([][]*net.IPNet{{ipNet1}, {ipNet2}})

	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}

	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"172.92.0.1/32", "fe80::1/128"}
	conn, err := srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn, []string{"172.92.0.0/32", "fe80::/128"}, []string{"172.92.0.2/32", "fe80::2/128"})

	req.Connection = conn
	conn, err = srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn, []string{"172.92.0.0/32", "fe80::/128"}, []string{"172.92.0.2/32", "fe80::2/128"})

	req.Connection = conn.Clone()
	req.Connection.Context.IpContext.ExcludedPrefixes = []string{"172.92.0.1/30", "fe80::1/126"}
	conn, err = srv.Request(context.Background(), req)
	require.NoError(t, err)
	requireConns(t, conn, []string{"172.92.0.4/32", "fe80::4/128"}, []string{"172.92.0.5/32", "fe80::5/128"})
}

func Test_NewServer_GroupOfCustomIPAMServers(t *testing.T) {
	requireConns := func(t *testing.T, conn *networkservice.Connection, srcs []string) {
		require.Equal(t, len(srcs), len(conn.GetContext().GetIpContext().GetSrcIpAddrs()))
		for i, src := range srcs {
			require.Equal(t, src, conn.GetContext().GetIpContext().GetSrcIpAddrs()[i])
		}
	}

	_, ipNet1, err := net.ParseCIDR("172.92.3.4/16")
	require.NoError(t, err)
	_, ipNet2, err := net.ParseCIDR("fd00::/8")
	require.NoError(t, err)

	srv := chain.NewNetworkServiceServer(
		updatepath.NewServer("ipam"),
		metadata.NewServer(),
		groupipam.NewServer([][]*net.IPNet{{ipNet1}, {ipNet2}}, groupipam.WithCustomIPAMServer(singlepointipam.NewServer)),
	)
	req := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}

	conn1, err := srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn1, []string{"172.92.0.1/16", "fd00::1/8"})

	conn2, err := srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn2, []string{"172.92.0.2/16", "fd00::2/8"})

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn3, err := srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn3, []string{"172.92.0.1/16", "fd00::1/8"})

	conn4, err := srv.Request(context.Background(), req.Clone())
	require.NoError(t, err)
	requireConns(t, conn4, []string{"172.92.0.3/16", "fd00::3/8"})
}

func TestOutOfIPs(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("192.168.1.2/31")
	require.NoError(t, err)

	srv1 := groupipam.NewServer([][]*net.IPNet{{ipNet}})
	srv2 := groupipam.NewServer([][]*net.IPNet{{ipNet}})

	req1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.NewString(),
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}

	req2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: uuid.NewString(),
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}
	for i := 0; i < 100; i++ {
		conn1, err := srv1.Request(context.Background(), req1)
		require.NoError(t, err)
		requireConns(t, conn1, "192.168.1.2/32", "192.168.1.3/32")
		req1.Connection = conn1

		conn2, err := srv2.Request(context.Background(), req2)
		require.NoError(t, err)
		requireConns(t, conn2, "192.168.1.2/32", "192.168.1.3/32")
		req2.Connection = conn2

		_, err = srv1.Request(context.Background(), req2)
		require.Error(t, err)

		_, err = srv2.Request(context.Background(), req1)
		require.Error(t, err)

		_, err = srv2.Close(context.Background(), req2.GetConnection())
		require.NoError(t, err)
		_, err = srv1.Close(context.Background(), req1.GetConnection())
		require.NoError(t, err)
	}
}

func requireConns(t *testing.T, conn *networkservice.Connection, dstAddr, srcAddr string) {
	for i, src := range conn.GetContext().GetIpContext().GetSrcIpAddrs() {
		require.Equal(t, srcAddr, src, i)
	}
	for i, dst := range conn.GetContext().GetIpContext().GetDstIpAddrs() {
		require.Equal(t, dstAddr, dst, i)
	}
}
