// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/setid"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"

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
			require.NotNil(t, clienturlctx.ClientURL(ctx))
		}),
	)
	_, err = server.Request(context.Background(), request)
	require.Nil(t, err)
}

func TestNoMatchServiceFound(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()

	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())
	for _, nse := range endpoints() {
		_, err := nseServer.Register(context.Background(), nse)
		require.Nil(t, err)
	}
	want := labels(nsName,
		map[string]string{
			"app": "firewall",
		})
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Payload:        payload.IP,
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()
	_, err := server.Request(ctx, request)
	require.Error(t, err)
}

func TestNoMatchServiceEndpointFound(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	nsName := networkServiceName()
	nsServer := memory.NewNetworkServiceRegistryServer()
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name:    nsName,
		Matches: []*registry.Match{fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch()},
	})
	require.Nil(t, err)

	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(setid.NewNetworkServiceEndpointRegistryServer(), memory.NewNetworkServiceEndpointRegistryServer())

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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()
	_, err = server.Request(ctx, request)
	require.Error(t, err)
}

func TestMatchExactService(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	nsServer := memory.NewNetworkServiceRegistryServer()
	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(
		setid.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	nsName := networkServiceName()
	server := next.NewNetworkServiceServer(
		discover.NewServer(
			adapters.NetworkServiceServerToClient(nsServer),
			adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, nsName, nses[0].NetworkServiceNames[0])
		}),
	)

	// 1. Register NS, NSE with wrong name
	wrongNSName := nsName + "-wrong"
	_, err := nsServer.Register(context.Background(), &registry.NetworkService{
		Name: wrongNSName,
	})
	require.NoError(t, err)
	_, err = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{wrongNSName},
	})
	require.NoError(t, err)

	// 2. Try to discover NSE by the right NS name
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
		},
	}

	_, err = server.Request(ctx, request.Clone())
	require.Error(t, err)

	// 3. Register NS, NSE with the right name
	_, err = nsServer.Register(context.Background(), &registry.NetworkService{
		Name: nsName,
	})
	require.NoError(t, err)
	_, err = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{nsName},
	})
	require.NoError(t, err)

	// 4. Try to discover NSE by the right NS name
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = server.Request(ctx, request.Clone())
	require.NoError(t, err)
}

func TestMatchExactEndpoint(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(
		setid.NewNetworkServiceEndpointRegistryServer(),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	nseName := "final-endpoint"
	u := "tcp://" + nseName
	server := next.NewNetworkServiceServer(
		discover.NewServer(
			adapters.NetworkServiceServerToClient(memory.NewNetworkServiceRegistryServer()),
			adapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.Equal(t, u, clienturlctx.ClientURL(ctx).String())
		}),
	)

	// 1. Register NSE with wrong name
	wrongNSEName := nseName + "-wrong"
	wrongURL := u + "-wrong"
	_, err := nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: wrongNSEName,
		Url:  wrongURL,
	})
	require.NoError(t, err)

	// 2. Try to discover NSE by the right name
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: nseName,
		},
	}

	_, err = server.Request(ctx, request.Clone())
	require.Error(t, err)

	// 3. Register NSE with the right name
	_, err = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: nseName,
		Url:  u,
	})
	require.NoError(t, err)

	// 4. Try to discover NSE by the right name
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err = server.Request(ctx, request.Clone())
	require.NoError(t, err)
}
