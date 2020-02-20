// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// package discover_test contains tests for package 'discover'
package discover_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
)

type testNetworkServiceServer struct {
	t    *testing.T
	want []*registry.NetworkServiceEndpoint
}

func (s testNetworkServiceServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	assert.Equal(s.t, s.want, discover.Candidates(ctx).NetworkServiceEndpoints)
	return in.GetConnection(), nil
}

func (s testNetworkServiceServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

type mockNetworkServiceDiscoveryClient struct {
	response registry.FindNetworkServiceResponse
}

func (c *mockNetworkServiceDiscoveryClient) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest, ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	return &c.response, nil
}

var discoveryClientResponse = &registry.FindNetworkServiceResponse{
	NetworkService: &registry.NetworkService{
		Name: "secure-intranet-connectivity",
		Matches: []*registry.Match{
			{
				SourceSelector: map[string]string{
					"app": "firewall",
				},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"app": "vpn-gateway",
						},
					},
				},
			},
			{
				SourceSelector: map[string]string{},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"app": "firewall",
						},
					},
				},
			},
		},
	},
	NetworkServiceEndpoints: []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
		{
			Labels: map[string]string{
				"app": "vpn-gateway",
			},
		},
	},
}

var testData = []struct {
	name    string
	request *networkservice.NetworkServiceRequest
	want    []*registry.NetworkServiceEndpoint
}{
	{
		name: "with empty source selector",
		request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				NetworkService: "secure-intranet-connectivity",
				Labels:         map[string]string{},
			},
		},
		want: []*registry.NetworkServiceEndpoint{
			{
				Labels: map[string]string{
					"app": "firewall",
				},
			},
		},
	},
	{
		name: "with not empty source selector",
		request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				NetworkService: "secure-intranet-connectivity",
				Labels: map[string]string{
					"app": "firewall",
				},
			},
		},
		want: []*registry.NetworkServiceEndpoint{
			{
				Labels: map[string]string{
					"app": "vpn-gateway",
				},
			},
		},
	},
}

func Test_discoverServer_Request(t *testing.T) {
	for _, data := range testData {
		test := data
		t.Run(test.name, func(t *testing.T) {
			testRequest(t, test.request, test.want)
		})
	}
}

func testRequest(t *testing.T, request *networkservice.NetworkServiceRequest, want []*registry.NetworkServiceEndpoint) {
	discoveryClient := &mockNetworkServiceDiscoveryClient{*discoveryClientResponse}
	server := next.NewNetworkServiceServer(discover.NewServer(discoveryClient), &testNetworkServiceServer{t, want})
	_, _ = server.Request(context.Background(), request)
}
