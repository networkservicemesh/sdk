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

package excludedprefixes_test

import (
	"context"
	"net"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectexcludedprefixes"
)

func TestExcludedPrefixesClient_Request_PrefixesAreDifferent(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := excludedprefixes.NewClient()

	srcCidr := "172.16.0.100/30"
	_, ipNet, err := net.ParseCIDR(srcCidr)
	require.NoError(t, err)

	server1 := adapters.NewServerToClient(
		point2pointipam.NewServer(ipNet),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(ctx, request.Clone())
	require.NoError(t, err)

	expectedExcludedIPs := append(resp.GetContext().GetIpContext().GetSrcIpAddrs(),
		resp.GetContext().GetIpContext().GetDstIpAddrs()...)

	destIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, destIPs, 1)
	firstSrcIP := destIPs[0]

	server2 := adapters.NewServerToClient(
		point2pointipam.NewServer(ipNet),
	)

	resp, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request.Clone())
	require.NoError(t, err)

	destIPs = resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, destIPs, 1)
	secondSrcIP := destIPs[0]

	require.NotEqual(t, firstSrcIP, secondSrcIP)

	excludedIPs := resp.GetContext().GetIpContext().GetExcludedPrefixes()
	require.Equal(t, excludedIPs, expectedExcludedIPs)
}

func TestExcludedPrefixesClient_Close_PrefixesAreEmpty(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := excludedprefixes.NewClient()

	firstRequest := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"172.16.0.100/32"},
					DstIpAddrs: []string{"172.16.0.103/32"},
				},
			},
		},
	}

	secondRequest := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	firstResp, err := client.Request(ctx, firstRequest.Clone())
	require.NoError(t, err)

	expectedExcludedIPs := append(firstRequest.GetConnection().GetContext().GetIpContext().GetSrcIpAddrs(),
		firstRequest.GetConnection().GetContext().GetIpContext().GetDstIpAddrs()...)

	secondResp, err := client.Request(ctx, secondRequest.Clone())
	require.NoError(t, err)

	excludedIPs := secondResp.GetContext().GetIpContext().GetExcludedPrefixes()
	require.Equal(t, excludedIPs, expectedExcludedIPs)

	_, err = client.Close(ctx, firstResp)
	require.NoError(t, err)

	secondResp, err = client.Request(ctx, secondRequest.Clone())
	require.NoError(t, err)

	require.Empty(t, secondResp.GetContext().GetIpContext().GetExcludedPrefixes())

	_, err = client.Close(ctx, secondResp)
	require.NoError(t, err)

	require.Empty(t, secondResp.GetContext().GetIpContext().GetExcludedPrefixes())
}

func TestExcludedPrefixesClient_Request_WithExcludedPrefixes(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.96/32", "172.16.0.98/32", "172.16.0.100"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(ctx, excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(context.Background(), request.Clone())
	require.NoError(t, err)

	// sanity checks
	possibleIPs := []string{"172.16.0.97/32", "172.16.0.99/32", "172.16.0.101/32", "172.16.0.103/32"}
	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)
	require.Contains(t, possibleIPs, srcIPs[0])

	destIPs := resp.GetContext().GetIpContext().GetDstIpAddrs()
	require.Len(t, destIPs, 1)
	require.Contains(t, possibleIPs, destIPs[0])

	require.NotEqual(t, srcIPs[0], destIPs[0])

	expectedExcludedPrefixes := make([]string, 0)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, srcIPs...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, destIPs...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, excludedPrefixes...)

	// second request
	server2 := adapters.NewServerToClient(
		point2pointipam.NewServer(ipNet),
	)

	resp, err = chain.NewNetworkServiceClient(client, server2).Request(context.Background(), request.Clone())
	require.NoError(t, err)

	require.Equal(t, resp.GetContext().GetIpContext().GetExcludedPrefixes(), expectedExcludedPrefixes)
}
