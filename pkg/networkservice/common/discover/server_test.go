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

	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
)

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
	defer goleak.VerifyNone(t)
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	want := []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels:         map[string]string{},
		},
	}
	server := next.NewNetworkServiceServer(
		discover.NewServer(discoveryClient),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			assert.Equal(t, want, discover.Candidates(ctx).GetNetworkServiceEndpoints())
		}),
	)
	_, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
}

func TestMatchNonEmptySourceSelector(t *testing.T) {
	defer goleak.VerifyNone(t)
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
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
	server := next.NewNetworkServiceServer(
		discover.NewServer(discoveryClient),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			assert.Equal(t, want, discover.Candidates(ctx).GetNetworkServiceEndpoints())
		}),
	)
	_, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
}

func TestMatchEmptySourceSelectorGoingFirst(t *testing.T) {
	defer goleak.VerifyNone(t)
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	want := []*registry.NetworkServiceEndpoint{
		{
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	server := next.NewNetworkServiceServer(
		discover.NewServer(discoveryClient),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			assert.Equal(t, want, discover.Candidates(ctx).GetNetworkServiceEndpoints())
		}),
	)
	_, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
}

func TestMatchNothing(t *testing.T) {
	defer goleak.VerifyNone(t)
	registryResponse := &registry.FindNetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name:    "secure-intranet-connectivity",
			Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch()},
		},
		NetworkServiceEndpoints: endpoints(),
	}
	discoveryClient := &mockNetworkServiceDiscoveryClient{registryResponse}
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "unknown-app",
			},
		},
	}
	want := endpoints()
	server := next.NewNetworkServiceServer(
		discover.NewServer(discoveryClient),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			assert.Equal(t, want, discover.Candidates(ctx).GetNetworkServiceEndpoints())
		}),
	)
	_, err := server.Request(context.Background(), request)
	assert.Nil(t, err)
}
