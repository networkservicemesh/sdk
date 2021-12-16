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
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/ipam/point2pointipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectexcludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectipam"
)

func TestExcludedPrefixesClient_Request_SanityCheck(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	excludedPrefixes := []string{"172.16.0.96/32", "172.16.0.98/32", "172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(context.Background(), request.Clone())
	require.NoError(t, err)

	possibleIPs := []string{"172.16.0.97/32", "172.16.0.99/32", "172.16.0.101/32", "172.16.0.103/32"}
	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)
	require.Contains(t, possibleIPs, srcIPs[0])

	destIPs := resp.GetContext().GetIpContext().GetDstIpAddrs()
	require.Len(t, destIPs, 1)
	require.Contains(t, possibleIPs, destIPs[0])

	require.NotEqual(t, srcIPs[0], destIPs[0])
}

func TestExcludedPrefixesClient_Request_SrcAndDestPrefixesAreDifferent(t *testing.T) {
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

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(ctx, request1)
	require.NoError(t, err)

	expectedExcludedIPs := append(resp.GetContext().GetIpContext().GetSrcIpAddrs(),
		resp.GetContext().GetIpContext().GetDstIpAddrs()...)
	expectedExcludedIPs = append(expectedExcludedIPs, getPrefixes(resp.GetContext().GetIpContext().GetSrcRoutes())...)
	expectedExcludedIPs = append(expectedExcludedIPs, getPrefixes(resp.GetContext().GetIpContext().GetDstRoutes())...)
	expectedExcludedIPs = excludedprefixes.RemoveDuplicates(expectedExcludedIPs)

	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)
	srcIP1 := srcIPs[0]

	server2 := adapters.NewServerToClient(
		point2pointipam.NewServer(ipNet),
	)

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request2)
	require.NoError(t, err)

	srcIPs = resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)
	srcIP2 := srcIPs[0]

	require.NotEqual(t, srcIP1, srcIP2)

	excludedIPs := resp.GetContext().GetIpContext().GetExcludedPrefixes()
	require.ElementsMatch(t, expectedExcludedIPs, excludedIPs)
}

func TestExcludedPrefixesClient_Close_PrefixesAreRemoved(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx := context.Background()

	client := excludedprefixes.NewClient()

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: &networkservice.IPContext{
					SrcIpAddrs: []string{"172.16.0.100/32"},
					DstIpAddrs: []string{"172.16.0.103/32"},
				},
			},
		},
	}

	resp1, err := client.Request(ctx, request1)
	require.NoError(t, err)

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	_, err = client.Close(ctx, resp1)
	require.NoError(t, err)

	respCheckEmpty, err := client.Request(ctx, request2)
	require.NoError(t, err)

	require.Empty(t, respCheckEmpty.GetContext().GetIpContext().GetExcludedPrefixes())
}

func TestExcludedPrefixesClient_Request_WithExcludedPrefixes(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.96/32", "172.16.0.98/32", "172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(ctx, request1)
	require.NoError(t, err)

	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)

	destIPs := resp.GetContext().GetIpContext().GetDstIpAddrs()
	require.Len(t, destIPs, 1)

	expectedExcludedPrefixes := make([]string, 0)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, srcIPs[0], destIPs[0])
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetSrcRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetDstRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, excludedPrefixes...)
	expectedExcludedPrefixes = excludedprefixes.RemoveDuplicates(expectedExcludedPrefixes)

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	server2 := adapters.NewServerToClient(
		point2pointipam.NewServer(ipNet),
	)

	resp, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request2)
	require.NoError(t, err)

	require.ElementsMatch(t, expectedExcludedPrefixes, resp.GetContext().GetIpContext().GetExcludedPrefixes())
}

func TestExcludedPrefixesClient_Request_PrefixesUnchangedAfterError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.96/32", "172.16.0.98/32", "172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request1 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(ctx, request1)
	require.NoError(t, err)

	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)

	destIPs := resp.GetContext().GetIpContext().GetDstIpAddrs()
	require.Len(t, destIPs, 1)

	expectedExcludedPrefixes := make([]string, 0)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, srcIPs[0], destIPs[0])
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetSrcRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetDstRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, excludedPrefixes...)
	expectedExcludedPrefixes = excludedprefixes.RemoveDuplicates(expectedExcludedPrefixes)

	// request with error
	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	server2 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
		injecterror.NewClient(injecterror.WithError(errors.Errorf("Test error"))),
	)

	_, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request2)
	require.Error(t, err)

	// third request to get the final prefixes
	request3 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	server3 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	resp, err = chain.NewNetworkServiceClient(client, server3).Request(ctx, request3)
	require.NoError(t, err)

	require.ElementsMatch(t, expectedExcludedPrefixes, resp.GetContext().GetIpContext().GetExcludedPrefixes())
}

func TestExcludedPrefixesClient_Request_SuccessfulRefresh(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.96/32", "172.16.0.98/32", "172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err := chain.NewNetworkServiceClient(client, server1).Request(ctx, request)
	require.NoError(t, err)

	srcIPs := resp.GetContext().GetIpContext().GetSrcIpAddrs()
	require.Len(t, srcIPs, 1)

	destIPs := resp.GetContext().GetIpContext().GetDstIpAddrs()
	require.Len(t, destIPs, 1)

	// expected excluded prefixes for refresh use-case
	// src/dest IPs won't be present in IPContext
	expectedExcludedPrefixes := make([]string, 0)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetSrcRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, getPrefixes(resp.GetContext().GetIpContext().GetDstRoutes())...)
	expectedExcludedPrefixes = append(expectedExcludedPrefixes, excludedPrefixes...)
	expectedExcludedPrefixes = excludedprefixes.RemoveDuplicates(expectedExcludedPrefixes)
	expectedExcludedPrefixes = excludedprefixes.Exclude(expectedExcludedPrefixes, []string{srcIPs[0], destIPs[0]})

	// expected excluded prefixes for a non-refresh use-case
	// src/dest IPs from first server should still be present in a request to another server
	expectedExcludedPrefixes2 := make([]string, 0)
	expectedExcludedPrefixes2 = append(expectedExcludedPrefixes2, srcIPs[0], destIPs[0])
	expectedExcludedPrefixes2 = append(expectedExcludedPrefixes2, getPrefixes(resp.GetContext().GetIpContext().GetSrcRoutes())...)
	expectedExcludedPrefixes2 = append(expectedExcludedPrefixes2, getPrefixes(resp.GetContext().GetIpContext().GetDstRoutes())...)
	expectedExcludedPrefixes2 = append(expectedExcludedPrefixes2, excludedPrefixes...)
	expectedExcludedPrefixes2 = excludedprefixes.RemoveDuplicates(expectedExcludedPrefixes2)

	resp, err = chain.NewNetworkServiceClient(client, server1).Request(ctx, request)
	require.NoError(t, err)

	require.ElementsMatch(t, expectedExcludedPrefixes, resp.GetContext().GetIpContext().GetExcludedPrefixes())

	server2 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request2 := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	resp, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request2)
	require.NoError(t, err)

	require.ElementsMatch(t, expectedExcludedPrefixes2, resp.GetContext().GetIpContext().GetExcludedPrefixes())
}

func TestExcludedPrefixesClient_Request_EndpointConflicts(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	_, err = chain.NewNetworkServiceClient(client, server1).Request(ctx, request.Clone())
	require.NoError(t, err)

	// conflict with existing routes (172.16.0.96/32)
	server2 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(injectipam.NewServer(
			[]string{"172.16.0.101/32"},
			[]string{"172.16.0.96/32"},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.101/32",
					NextHop: "",
				},
			},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.96/32",
					NextHop: "",
				},
			})))

	_, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request.Clone())
	require.Error(t, err)

	// conflict with already excluded prefixes (172.16.0.100/32)
	server3 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(injectipam.NewServer(
			[]string{"172.16.0.100/32"},
			[]string{"172.16.0.103/32"},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.103/32",
					NextHop: "",
				},
			},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.100/32",
					NextHop: "",
				},
			})))

	_, err = chain.NewNetworkServiceClient(client, server3).Request(ctx, request.Clone())
	require.Error(t, err)
}

func TestExcludedPrefixesClient_Request_EndpointConflictCloseError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	client := excludedprefixes.NewClient()

	_, ipNet, err := net.ParseCIDR("172.16.0.96/29")
	require.NoError(t, err)

	ctx := context.Background()

	excludedPrefixes := []string{"172.16.0.100/32"}

	server1 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(
			injectexcludedprefixes.NewServer(excludedPrefixes)),
		adapters.NewServerToClient(
			point2pointipam.NewServer(ipNet)),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{},
	}

	_, err = chain.NewNetworkServiceClient(client, server1).Request(ctx, request.Clone())
	require.NoError(t, err)

	// conflict with existing routes (172.16.0.96/32)
	server2 := chain.NewNetworkServiceClient(
		adapters.NewServerToClient(injectipam.NewServer(
			[]string{"172.16.0.101/32"},
			[]string{"172.16.0.96/32"},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.101/32",
					NextHop: "",
				},
			},
			[]*networkservice.Route{
				{
					Prefix:  "172.16.0.96/32",
					NextHop: "",
				},
			})),
		injecterror.NewClient(injecterror.WithCloseErrorTimes(-1), injecterror.WithRequestErrorTimes(-2)))

	_, err = chain.NewNetworkServiceClient(client, server2).Request(ctx, request.Clone())
	require.Error(t, err)
	require.Contains(t, err.Error(), "connection closed")
}

func getPrefixes(routes []*networkservice.Route) []string {
	var rv []string
	for _, route := range routes {
		rv = append(rv, route.GetPrefix())
	}
	return rv
}
