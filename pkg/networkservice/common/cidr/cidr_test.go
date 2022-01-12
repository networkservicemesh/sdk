// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

func TestIPFamilies(t *testing.T) {
	var samples = []struct {
		name      string
		family    networkservice.IpFamily_Family
		prefixLen uint32
		prefix    string
		cidr0     string
		cidr1     string
		cidr2     string
	}{
		{
			name:      "IPv4",
			family:    networkservice.IpFamily_IPV4,
			prefix:    "192.168.0.0/16",
			prefixLen: 24,
			cidr0:     "192.168.0.0/24",
			cidr1:     "192.168.1.0/24",
			cidr2:     "192.168.2.0/24",
		},
		{
			name:      "IPv6",
			family:    networkservice.IpFamily_IPV6,
			prefix:    "2001:db8::/96",
			prefixLen: 112,
			cidr0:     "2001:db8::/112",
			cidr1:     "2001:db8::1:0/112",
			cidr2:     "2001:db8::2:0/112",
		},
	}

	for _, sample := range samples {
		t.Run(sample.name, func(t *testing.T) {
			// nolint:scopelint
			testIPFamilies(t, sample.family, sample.prefixLen, sample.prefix, sample.cidr0, sample.cidr1, sample.cidr2)
		})
	}
}

func testIPFamilies(t *testing.T, family networkservice.IpFamily_Family, prefixLen uint32, prefix, cidr0, cidr1, cidr2 string) {
	prefixes := []string{prefix}
	server := newServer(prefixes, nil)

	request := newRequest()
	client1 := newClient(prefixLen, family, server)
	conn1, err := client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, cidr0, prefix)

	// refresh
	conn1, err = client1.Request(context.Background(), request)
	require.NoError(t, err)
	validateConn(t, conn1, cidr0, prefix)

	client2 := newClient(prefixLen, family, server)
	conn2, err := client2.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn2, cidr1, prefix)

	_, err = client1.Close(context.Background(), conn1)
	require.NoError(t, err)

	client3 := newClient(prefixLen, family, server)
	conn3, err := client3.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn3, cidr0, prefix)

	client4 := newClient(prefixLen, family, server)
	conn4, err := client4.Request(context.Background(), newRequest())
	require.NoError(t, err)
	validateConn(t, conn4, cidr2, prefix)
}

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
