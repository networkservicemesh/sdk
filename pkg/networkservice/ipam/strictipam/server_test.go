// Copyright (c) 2024 Cisco and/or its affiliates.
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

package strictipam_test

import (
	"context"
	"net"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/strictipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkrequest"
)

func newRequest(connID string) *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: connID,
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}
}
func validateConns(t *testing.T, conn *networkservice.Connection, dsts, srcs []string) {
	for i, dst := range dsts {
		require.Equal(t, conn.Context.IpContext.DstIpAddrs[i], dst)
		require.Equal(t, conn.Context.IpContext.SrcRoutes[i].Prefix, dst)
	}
	for i, src := range srcs {
		require.Equal(t, conn.Context.IpContext.SrcIpAddrs[i], src)
		require.Equal(t, conn.Context.IpContext.DstRoutes[i].Prefix, src)
	}
}

// nolint: dupl
func TestOverlappingAddresses(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("172.16.0.0/24")
	require.NoError(t, err)

	srv := next.NewNetworkServiceServer(strictipam.NewServer(point2pointipam.NewServer, ipNet))

	emptyRequest := newRequest("empty")

	request := newRequest("id")
	request.Connection.Context.IpContext.SrcIpAddrs = []string{"172.16.0.1/32", "172.16.0.25/32"}
	request.Connection.Context.IpContext.DstIpAddrs = []string{"172.16.0.0/32", "172.16.0.24/32"}
	request.Connection.Context.IpContext.SrcRoutes = []*networkservice.Route{{Prefix: "172.16.0.0/32"}, {Prefix: "172.16.0.24/32"}}
	request.Connection.Context.IpContext.DstRoutes = []*networkservice.Route{{Prefix: "172.16.0.1/32"}, {Prefix: "172.16.0.25/32"}}

	conn1, err := srv.Request(context.Background(), emptyRequest)
	require.NoError(t, err)
	validateConns(t, conn1, []string{"172.16.0.0/32"}, []string{"172.16.0.1/32"})

	conn2, err := srv.Request(context.Background(), request.Clone())
	require.NoError(t, err)
	validateConns(t, conn2, []string{"172.16.0.24/32"}, []string{"172.16.0.25/32"})

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn2, err = srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConns(t, conn2, []string{"172.16.0.0/32", "172.16.0.24/32"}, []string{"172.16.0.1/32", "172.16.0.25/32"})
}

// nolint: dupl
func TestOverlappingAddressesIPv6(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("fe80::/64")
	require.NoError(t, err)

	srv := next.NewNetworkServiceServer(strictipam.NewServer(point2pointipam.NewServer, ipNet))

	emptyRequest := newRequest("empty")

	request := newRequest("id")
	request.Connection.Id = "id"
	request.Connection.Context.IpContext.SrcIpAddrs = []string{"fe80::1/128", "fe80::fa01/128"}
	request.Connection.Context.IpContext.DstIpAddrs = []string{"fe80::/128", "fe80::fa00/128"}
	request.Connection.Context.IpContext.SrcRoutes = []*networkservice.Route{{Prefix: "fe80::/128"}, {Prefix: "fe80::fa00/128"}}
	request.Connection.Context.IpContext.DstRoutes = []*networkservice.Route{{Prefix: "fe80::1/128"}, {Prefix: "fe80::fa01/128"}}

	conn1, err := srv.Request(context.Background(), emptyRequest)
	require.NoError(t, err)
	validateConns(t, conn1, []string{"fe80::/128"}, []string{"fe80::1/128"})

	conn2, err := srv.Request(context.Background(), request.Clone())
	require.NoError(t, err)
	validateConns(t, conn2, []string{"fe80::fa00/128"}, []string{"fe80::fa01/128"})

	_, err = srv.Close(context.Background(), conn1)
	require.NoError(t, err)

	conn2, err = srv.Request(context.Background(), request)
	require.NoError(t, err)
	validateConns(t, conn2, []string{"fe80::/128", "fe80::fa00/128"}, []string{"fe80::1/128", "fe80::fa01/128"})
}

func Test_StrictIPAM_PositiveScenario(t *testing.T) {
	_, ipNet, err := net.ParseCIDR("172.16.1.0/29")
	require.NoError(t, err)

	var s = strictipam.NewServer(func(i ...*net.IPNet) networkservice.NetworkServiceServer {
		return chain.NewNetworkServiceServer(
			checkrequest.NewServer(t, func(t *testing.T, nsr *networkservice.NetworkServiceRequest) {
				require.NotEqual(t, networkservice.IPContext{}, *nsr.GetConnection().Context.GetIpContext(), "ip context should not be empty")
			}),
			point2pointipam.NewServer(ipNet))
	}, ipNet)

	_, err = s.Request(context.TODO(), &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"172.16.1.0/32"},
				},
			},
		},
	})
	require.NoError(t, err)
}
