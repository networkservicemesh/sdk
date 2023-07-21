// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"time"

	"github.com/networkservicemesh/api/pkg/api/ipam"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_NSC_ConnectsToVl3NSE(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

	var prefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(prefixCh)

	prefixCh <- &ipam.PrefixResponse{Prefix: "10.0.0.1/24"}

	var server = next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), prefixCh),
	)

	require.Eventually(t, func() bool { return len(prefixCh) == 0 }, time.Second, time.Millisecond*100)

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
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

	var prefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(prefixCh)

	prefixCh <- &ipam.PrefixResponse{Prefix: "12.0.0.1/24"}

	var server = next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), prefixCh),
	)

	require.Eventually(t, func() bool { return len(prefixCh) == 0 }, time.Second, time.Millisecond*120)

	resp, err := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))

	require.NoError(t, err)

	require.Equal(t, "12.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "12.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "12.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "12.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "12.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "12.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())

	prefixCh <- &ipam.PrefixResponse{Prefix: "11.0.0.1/24"}
	require.Eventually(t, func() bool { return len(prefixCh) == 0 }, time.Second, time.Millisecond*100)

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
	t.Parallel()
	t.Cleanup(func() { goleak.VerifyNone(t, goleak.IgnoreCurrent()) })

	var prefixCh = make(chan *ipam.PrefixResponse, 1)
	defer close(prefixCh)

	prefixCh <- &ipam.PrefixResponse{Prefix: "10.0.0.1/24"}

	var server = next.NewNetworkServiceServer(
		metadata.NewServer(),
		vl3.NewServer(context.Background(), prefixCh),
	)

	require.Eventually(t, func() bool { return len(prefixCh) == 0 }, time.Second, time.Millisecond*100)

	for i := 0; i < 10; i++ {
		resp, err := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))

		require.NoError(t, err)

		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0], i)
		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0], i)

		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix(), i)
		require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix(), i)
		require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix(), i)

		resp1, err1 := server.Request(context.Background(), new(networkservice.NetworkServiceRequest))

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
