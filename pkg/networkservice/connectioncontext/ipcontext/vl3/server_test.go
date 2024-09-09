// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

package vl3_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/ipam/strictvl3ipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_NSC_ConnectsToVl3NSE(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})
	ipam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), ipam),
	)

	resp, err := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))

	require.NoError(t, err)

	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())

	for i := 0; i < 10; i++ {
		resp, err = server.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: resp})

		require.NoError(t, err)

		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
		require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
		require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())

		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])
	}
}

func Test_NSC_ConnectsToVl3NSE_PrefixHasChanged(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ipam := vl3.NewIPAM("12.0.0.1/24")

	server := next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), ipam),
	)

	resp, err := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))

	require.NoError(t, err)

	require.Equal(t, "12.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "12.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "12.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "12.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "12.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "12.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())

	err = ipam.Reset("11.0.0.1/24")
	require.NoError(t, err)

	// refresh
	for i := 0; i < 10; i++ {
		resp, err = server.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: resp})

		require.NoError(t, err)

		require.Equal(t, "11.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
		require.Equal(t, "11.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

		require.Equal(t, "11.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
		require.Equal(t, "11.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
		require.Equal(t, "11.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
		require.Equal(t, "11.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	}
}

func Test_NSC_ConnectsToVl3NSE_Close(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ipam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), ipam),
	)

	for i := 0; i < 10; i++ {
		resp, err := server.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "1",
			},
		})

		require.NoError(t, err)

		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0], i)
		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0], i)

		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix(), i)
		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix(), i)

		resp1, err1 := server.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "2",
			},
		})

		require.NoError(t, err1)

		require.Equal(t, "10.0.0.2/32", resp1.GetContext().GetIpContext().GetSrcIpAddrs()[0], i)
		require.Equal(t, "10.0.0.0/32", resp1.GetContext().GetIpContext().GetDstIpAddrs()[0], i)

		require.Equal(t, "10.0.0.0/32", resp1.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/24", resp1.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/16", resp1.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix(), i)
		require.Equal(t, "10.0.0.2/32", resp1.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix(), i)

		_, err = server.Close(context.Background(), resp1)
		require.NoError(t, err, i)
		_, err = server.Close(context.Background(), resp)
		require.NoError(t, err, i)
	}
}

func Test_NSC_ConnectsToVl3NSE_DualStack(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	var ipams []*vl3.IPAM
	ipam1 := vl3.NewIPAM("10.0.0.1/24")
	ipams = append(ipams, ipam1)
	ipam2 := vl3.NewIPAM("2001:db8::/112")
	ipams = append(ipams, ipam2)

	server := next.NewNetworkServiceServer(
		metadata.NewServer(),
		strictvl3ipam.NewServer(context.Background(), vl3.NewServer, ipams...),
	)

	resp, err := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))
	require.NoError(t, err)
	ipContext := resp.GetContext().GetIpContext()

	require.Equal(t, "10.0.0.1/32", ipContext.GetSrcIpAddrs()[0])
	require.Equal(t, "2001:db8::1/128", ipContext.GetSrcIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", ipContext.GetDstIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", ipContext.GetDstIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", ipContext.GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", ipContext.GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", ipContext.GetSrcRoutes()[5].GetPrefix())
	require.Equal(t, "2001:db8::/128", ipContext.GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", ipContext.GetSrcRoutes()[3].GetPrefix())
	require.Equal(t, "2001:db8::/64", ipContext.GetSrcRoutes()[4].GetPrefix())

	require.Equal(t, "10.0.0.1/32", ipContext.GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "2001:db8::1/128", ipContext.GetDstRoutes()[1].GetPrefix())

	// refresh
	resp, err = server.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: resp})
	require.NoError(t, err)
	ipContext = resp.GetContext().GetIpContext()

	require.Equal(t, "10.0.0.1/32", ipContext.GetSrcIpAddrs()[0])
	require.Equal(t, "2001:db8::1/128", ipContext.GetSrcIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", ipContext.GetDstIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", ipContext.GetDstIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", ipContext.GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", ipContext.GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", ipContext.GetSrcRoutes()[5].GetPrefix())
	require.Equal(t, "2001:db8::/128", ipContext.GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", ipContext.GetSrcRoutes()[3].GetPrefix())
	require.Equal(t, "2001:db8::/64", ipContext.GetSrcRoutes()[4].GetPrefix())

	require.Equal(t, "10.0.0.1/32", ipContext.GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "2001:db8::1/128", ipContext.GetDstRoutes()[1].GetPrefix())
}
