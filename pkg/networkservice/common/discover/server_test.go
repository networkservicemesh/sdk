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

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"

	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	registrynext "github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func endpoints() []*registry.NetworkServiceEndpoint {
	ns := networkServiceName()
	return []*registry.NetworkServiceEndpoint{
		{
			NetworkServiceNames: []string{ns},
			NetworkServiceLabels: labels(ns,
				map[string]string{
					"app": "firewall",
				},
			),
		},
		{
			NetworkServiceNames: []string{ns},
			NetworkServiceLabels: labels(ns,
				map[string]string{
					"app": "some-middle-app",
				},
			),
		},
		{
			NetworkServiceNames: []string{ns},
			NetworkServiceLabels: labels(ns,
				map[string]string{
					"app": "vpn-gateway",
				},
			),
		},
	}
}

func networkServiceName() string {
	return "secure-intranet-connectivity"
}

func labels(service string, source map[string]string) map[string]*registry.NetworkServiceLabels {
	return map[string]*registry.NetworkServiceLabels{
		service: {
			Labels: source,
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
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
	})
	require.Nil(t, err)
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	for _, nse := range endpoints() {
		_, err = nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}
	want := labels(nsName,
		map[string]string{
			"app": "firewall",
		})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Labels:         map[string]string{},
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(adapters.NetworkServiceServerToClient(nsServer), adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].NetworkServiceLabels)
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}

func TestMatchNonEmptySourceSelector(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
	})
	require.Nil(t, err)
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	for _, nse := range endpoints() {
		_, err = nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	want := labels(nsName,
		map[string]string{
			"app": "some-middle-app",
		})
	server := next.NewNetworkServiceServer(
		discover.NewServer(adapters.NetworkServiceServerToClient(nsServer), adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].NetworkServiceLabels)
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}

func TestMatchEmptySourceSelectorGoingFirst(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch()},
	})
	require.Nil(t, err)
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	for _, nse := range endpoints() {
		_, err = nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}
	want := labels(nsName,
		map[string]string{
			"app": "firewall",
		})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Labels: map[string]string{
				"app": "firewall",
			},
		},
	}
	server := next.NewNetworkServiceServer(
		discover.NewServer(adapters.NetworkServiceServerToClient(nsServer), adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].NetworkServiceLabels)
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}

func TestMatchNothing(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch()},
	})
	require.Nil(t, err)
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	for _, nse := range endpoints() {
		_, err = nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "unknown-app",
			},
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(adapters.NetworkServiceServerToClient(nsServer), adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 3)
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}

func TestMatchSelectedNSE(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch()},
	})
	require.Nil(t, err)
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	var last *registry.NetworkServiceEndpoint
	for _, nse := range endpoints() {
		last, err = nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: last.Name,
		},
	}
	server := next.NewNetworkServiceServer(
		discover.NewServer(adapters.NetworkServiceServerToClient(nsServer), adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.NotNil(t, clienturl.ClientURL(ctx))
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}
