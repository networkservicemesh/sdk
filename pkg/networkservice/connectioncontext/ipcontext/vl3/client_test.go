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
	"time"

	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/ipam/strictvl3ipam"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/excludedprefixes"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/ipcontext/vl3"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_Client_ConnectsToVl3NSE(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ipam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(
			next.NewNetworkServiceClient(
				begin.NewClient(),
				metadata.NewClient(),
				excludedprefixes.NewClient(),
			),
		),
		metadata.NewServer(),
		vl3.NewServer(ctx, ipam),
	)

	resp, err := server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{Id: t.Name()}})

	require.NoError(t, err)

	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())

	// refresh
	resp, err = server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})

	require.NoError(t, err)

	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
}

func Test_VL3NSE_ConnectsToVl3NSE(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clientIpam := vl3.NewIPAM("10.0.1.0/24")
	serverIpam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(
			next.NewNetworkServiceClient(
				begin.NewClient(),
				metadata.NewClient(),
				vl3.NewClient(ctx, clientIpam),
			),
		),
		metadata.NewServer(),
		vl3.NewServer(ctx, serverIpam),
	)

	resp, err := server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{Id: t.Name()}})

	require.NoError(t, err)

	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.1.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())

	// refresh
	resp, err = server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})

	require.NoError(t, err)

	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.1.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())
}

func Test_VL3NSE_ConnectsToVl3NSE_DualStack(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var ipams []*vl3.IPAM
	ipam1 := vl3.NewIPAM("10.0.0.1/24")
	ipams = append(ipams, ipam1)
	ipam2 := vl3.NewIPAM("2001:db8::/112")
	ipams = append(ipams, ipam2)

	var clients []networkservice.NetworkServiceClient
	for _, ipam := range ipams {
		clients = append(clients, vl3.NewClient(ctx, ipam))
	}

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(
			next.NewNetworkServiceClient(
				begin.NewClient(),
				metadata.NewClient(),
				chain.NewNetworkServiceClient(clients...),
			),
		),
		metadata.NewServer(),
		strictvl3ipam.NewServer(ctx, vl3.NewServer, ipams...),
	)

	resp, err := server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{Id: t.Name()}})

	require.NoError(t, err)

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetSrcIpAddrs()[1])
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[2])
	require.Equal(t, "2001:db8::1/128", resp.GetContext().GetIpContext().GetSrcIpAddrs()[3])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetDstIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[5].GetPrefix())
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", resp.GetContext().GetIpContext().GetSrcRoutes()[3].GetPrefix())
	require.Equal(t, "2001:db8::/64", resp.GetContext().GetIpContext().GetSrcRoutes()[4].GetPrefix())

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetDstRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", resp.GetContext().GetIpContext().GetDstRoutes()[3].GetPrefix())
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[4].GetPrefix())
	require.Equal(t, "2001:db8::1/128", resp.GetContext().GetIpContext().GetDstRoutes()[5].GetPrefix())

	// refresh
	resp, err = server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})

	require.NoError(t, err)

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetSrcIpAddrs()[1])
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[2])
	require.Equal(t, "2001:db8::1/128", resp.GetContext().GetIpContext().GetSrcIpAddrs()[3])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetDstIpAddrs()[1])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[5].GetPrefix())
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", resp.GetContext().GetIpContext().GetSrcRoutes()[3].GetPrefix())
	require.Equal(t, "2001:db8::/64", resp.GetContext().GetIpContext().GetSrcRoutes()[4].GetPrefix())

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())
	require.Equal(t, "2001:db8::/128", resp.GetContext().GetIpContext().GetDstRoutes()[2].GetPrefix())
	require.Equal(t, "2001:db8::/112", resp.GetContext().GetIpContext().GetDstRoutes()[3].GetPrefix())
	require.Equal(t, "10.0.0.1/32", resp.GetContext().GetIpContext().GetDstRoutes()[4].GetPrefix())
	require.Equal(t, "2001:db8::1/128", resp.GetContext().GetIpContext().GetDstRoutes()[5].GetPrefix())
}

func Test_VL3NSE_ConnectsToVl3NSE_ChangePrefix(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clientIpam := vl3.NewIPAM("10.0.1.0/24")
	serverIpam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(
			next.NewNetworkServiceClient(
				begin.NewClient(),
				metadata.NewClient(),
				vl3.NewClient(ctx, clientIpam),
			),
		),
		metadata.NewServer(),
		vl3.NewServer(ctx, serverIpam),
	)

	resp, err := server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{Id: t.Name()}})

	require.NoError(t, err)

	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.1.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())

	err = clientIpam.Reset("10.0.5.0/24")
	require.NoError(t, err)

	// refresh
	for i := 0; i < 10; i++ {
		resp, err = server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: resp})

		require.NoError(t, err)

		require.Equal(t, "10.0.5.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[2])
		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

		require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
		require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
		require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
		require.Equal(t, "10.0.5.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[3].GetPrefix())
		require.Equal(t, "10.0.5.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[4].GetPrefix())
	}
}

func Test_VL3NSE_ConnectsToVl3NSE_Close(t *testing.T) {
	t.Cleanup(func() {
		goleak.VerifyNone(t)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clientIpam := vl3.NewIPAM("10.0.1.0/24")
	serverIpam := vl3.NewIPAM("10.0.0.1/24")

	server := next.NewNetworkServiceServer(
		adapters.NewClientToServer(
			next.NewNetworkServiceClient(
				begin.NewClient(),
				metadata.NewClient(),
				vl3.NewClient(ctx, clientIpam),
			),
		),
		metadata.NewServer(),
		vl3.NewServer(ctx, serverIpam),
	)

	resp, err := server.Request(ctx, &networkservice.NetworkServiceRequest{Connection: &networkservice.Connection{Id: uuid.New().String()}})

	require.NoError(t, err)

	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetSrcIpAddrs()[0])
	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetDstIpAddrs()[0])

	require.Equal(t, "10.0.0.0/32", resp.GetContext().GetIpContext().GetSrcRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.0.0/24", resp.GetContext().GetIpContext().GetSrcRoutes()[1].GetPrefix())
	require.Equal(t, "10.0.0.0/16", resp.GetContext().GetIpContext().GetSrcRoutes()[2].GetPrefix())
	require.Equal(t, "10.0.1.0/32", resp.GetContext().GetIpContext().GetDstRoutes()[0].GetPrefix())
	require.Equal(t, "10.0.1.0/24", resp.GetContext().GetIpContext().GetDstRoutes()[1].GetPrefix())

	_, err = server.Close(ctx, resp)

	require.NoError(t, err)
}
