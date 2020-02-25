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

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
)

type contextKeyType string

const (
	testDataKey contextKeyType = "testData"
)

type TestData struct {
	endpoints []*registry.NetworkServiceEndpoint
}

func withTestData(parent context.Context, testData *TestData) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, testDataKey, testData)
}

func testData(ctx context.Context) *TestData {
	if rv, ok := ctx.Value(testDataKey).(*TestData); ok {
		return rv
	}
	return nil
}

type testNetworkServiceServer struct {
}

func (s testNetworkServiceServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	testData(ctx).endpoints = discover.Candidates(ctx).GetNetworkServiceEndpoints()
	return next.Server(ctx).Request(ctx, in)
}

func (s testNetworkServiceServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

type mockNetworkServiceDiscoveryClient struct {
	response *registry.FindNetworkServiceResponse
}

func (c *mockNetworkServiceDiscoveryClient) FindNetworkService(context.Context, *registry.FindNetworkServiceRequest, ...grpc.CallOption) (*registry.FindNetworkServiceResponse, error) {
	return c.response, nil
}

func endpoints() []*registry.NetworkServiceEndpoint {
	return []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
		{
			Labels: map[string]string{
				"app": "some-middle-app",
			},
		},
		{
			Labels: map[string]string{
				"app": "vpn-gateway",
			},
		},
	}
}

func fromAnywhereMatch() *registry.Match {
	return &registry.Match{
		SourceSelector: map[string]string{},
		Routes: []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					"app": "firewall",
				},
			},
		},
	}
}

func fromFirewallMatch() *registry.Match {
	return &registry.Match{
		SourceSelector: map[string]string{
			"app": "firewall",
		},
		Routes: []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					"app": "some-middle-app",
				},
			},
		},
	}
}

func fromSomeMiddleAppMatch() *registry.Match {
	return &registry.Match{
		SourceSelector: map[string]string{
			"app": "some-middle-app",
		},
		Routes: []*registry.Destination{
			{
				DestinationSelector: map[string]string{
					"app": "vpn-gateway",
				},
			},
		},
	}
}

func TestMatchEmptySourceSelector(t *testing.T) {
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	server := next.NewNetworkServiceServer(discover.NewServer(discoveryClient), &testNetworkServiceServer{})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels:         map[string]string{},
		},
	}
	want := []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	ctx := withTestData(context.Background(), &TestData{})
	_, err := server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, want, testData(ctx).endpoints)
}

func TestMatchNonEmptySourceSelector(t *testing.T) {
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	server := next.NewNetworkServiceServer(discover.NewServer(discoveryClient), &testNetworkServiceServer{})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	want := []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "some-middle-app",
			},
		},
	}
	ctx := withTestData(context.Background(), &TestData{})
	_, err := server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, want, testData(ctx).endpoints)
}

func TestMatchEmptySourceSelectorGoingFirst(t *testing.T) {
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	server := next.NewNetworkServiceServer(discover.NewServer(discoveryClient), &testNetworkServiceServer{})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	want := []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	ctx := withTestData(context.Background(), &TestData{})
	_, err := server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, want, testData(ctx).endpoints)
}

func TestMatchNothing(t *testing.T) {
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	server := next.NewNetworkServiceServer(discover.NewServer(discoveryClient), &testNetworkServiceServer{})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "unknown-app",
			},
		},
	}
	want := endpoints()
	ctx := withTestData(context.Background(), &TestData{})
	_, err := server.Request(ctx, request)
	assert.Nil(t, err)
	assert.Equal(t, want, testData(ctx).endpoints)
}
