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

package cidr_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/cidr"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/stretchr/testify/require"
)

func newRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Context: &networkservice.ConnectionContext{
				IpContext: new(networkservice.IPContext),
			},
		},
	}
}

func newServer(prefixes, excludePrefixes []string) networkservice.NetworkServiceServer {
	return next.NewNetworkServiceServer(
		updatepath.NewServer("cidr-server"),
		metadata.NewServer(),
		cidr.NewServer(prefixes, excludePrefixes),
	)
}

func newClient(prefixLen uint32, family networkservice.IpFamily_Family, server networkservice.NetworkServiceServer) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		updatepath.NewClient("cidr-client"),
		metadata.NewClient(),
		cidr.NewClient(prefixLen, family),
		/* Server part */
		adapters.NewServerToClient(
			server,
		),
	)
}

func validateConn(t *testing.T, conn *networkservice.Connection, prefix, route string) {
	require.Equal(t, conn.Context.IpContext.ExtraPrefixes[0], prefix)
	require.Equal(t, conn.Context.IpContext.SrcRoutes, []*networkservice.Route{
		{
			Prefix: route,
		},
	})
}

//nolint:dupl
func TestIPv4(t *testing.T) {
	prefixes := []string{"192.168.0.0/16"}
	server := newServer(prefixes, nil)

	request := newRequest()
	client1 := newClient(24, networkservice.IpFamily_IPV4, server)
	conn1, err := client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/24", "192.168.0.0/16")

	// refresh
	conn1, err = client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "192.168.0.0/24", "192.168.0.0/16")

	client2 := newClient(24, networkservice.IpFamily_IPV4, server)
	conn2, err := client2.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "192.168.1.0/24", "192.168.0.0/16")

	_, err = client1.Close(context.Background(), conn1)
	require.NoError(t, err)

	client3 := newClient(24, networkservice.IpFamily_IPV4, server)
	conn3, err := client3.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "192.168.0.0/24", "192.168.0.0/16")

	client4 := newClient(24, networkservice.IpFamily_IPV4, server)
	conn4, err := client4.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "192.168.2.0/24", "192.168.0.0/16")
}

//nolint:dupl
func TestIPv6(t *testing.T) {
	prefixes := []string{"2001:db8::/96"}
	server := newServer(prefixes, nil)

	request := newRequest()
	client1 := newClient(112, networkservice.IpFamily_IPV6, server)
	conn1, err := client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "2001:db8::/112", "2001:db8::/96")

	// refresh
	conn1, err = client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, "2001:db8::/112", "2001:db8::/96")

	client2 := newClient(112, networkservice.IpFamily_IPV6, server)
	conn2, err := client2.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, "2001:db8::1:0/112", "2001:db8::/96")

	_, err = client1.Close(context.Background(), conn1)
	require.NoError(t, err)

	client3 := newClient(112, networkservice.IpFamily_IPV6, server)
	conn3, err := client3.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, "2001:db8::/112", "2001:db8::/96")

	client4 := newClient(112, networkservice.IpFamily_IPV6, server)
	conn4, err := client4.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, "2001:db8::2:0/112", "2001:db8::/96")
}

//nolint:dupl
func TestIPv4Exclude(t *testing.T) {
	prefixes := []string{"192.168.0.0/16"}
	excludePrefixes := []string{"192.168.0.0/24"}
	server := newServer(prefixes, excludePrefixes)

	client1 := newClient(24, networkservice.IpFamily_IPV4, server)
	conn1, err := client1.Request(context.Background(), newRequest())
	require.NoError(t, err)
	require.NotEqual(t, "192.168.0.0/24", conn1.Context.IpContext.ExtraPrefixes[0])
	require.Equal(t, conn1.Context.IpContext.SrcRoutes, []*networkservice.Route{
		{
			Prefix: "192.168.0.0/16",
		},
	})
}
