// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	registryadapters "github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	registrynext "github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

const (
	testWait = 100 * time.Millisecond
)

func endpoints() []*registry.NetworkServiceEndpoint {
	ns := networkServiceName()
	return []*registry.NetworkServiceEndpoint{
		{
			Name:                "nse-1",
			NetworkServiceNames: []string{ns},
			NetworkServiceLabels: labels(ns,
				map[string]string{
					"app": "firewall",
				},
			),
		},
		{
			Name:                "nse-2",
			NetworkServiceNames: []string{ns},
			NetworkServiceLabels: labels(ns,
				map[string]string{
					"app": "some-middle-app",
				},
			),
		},
		{
			Name:                "nse-3",
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

func testServers(
	t *testing.T,
	nsName string,
	nses []*registry.NetworkServiceEndpoint,
	matches ...*registry.Match,
) (registry.NetworkServiceRegistryServer, registry.NetworkServiceEndpointRegistryServer) {
	nsServer := memory.NewNetworkServiceRegistryServer()
	if nsName != "" {
		_, err := nsServer.Register(context.Background(), &registry.NetworkService{
			Name:    nsName,
			Matches: matches,
		})
		require.NoError(t, err)
	}

	nseServer := registrynext.NewNetworkServiceEndpointRegistryServer(
		memory.NewNetworkServiceEndpointRegistryServer(),
	)
	for i, nse := range nses {
		var err error
		nses[i], err = nseServer.Register(context.Background(), nse)
		require.NoError(t, err)
	}

	return nsServer, nseServer
}

func TestDiscoverCandidatesServer_MatchEmptySourceSelector(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, nsName, endpoints(), fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch())

	want := labels(nsName, map[string]string{
		"app": "firewall",
	})

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Labels:         map[string]string{},
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].GetNetworkServiceLabels())
		}),
	)

	_, err := server.Request(ctx, request)
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_MatchNonEmptySourceSelector(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, nsName, endpoints(), fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch())

	want := labels(nsName, map[string]string{
		"app": "some-middle-app",
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
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].GetNetworkServiceLabels())
		}),
	)

	_, err := server.Request(ctx, request)
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_MatchEmptySourceSelectorGoingFirst(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, nsName, endpoints(), fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch())

	want := labels(nsName, map[string]string{
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
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].GetNetworkServiceLabels())
		}),
	)

	_, err := server.Request(ctx, request)
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_MatchNothing(t *testing.T) {
	t.Skip()
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, nsName, endpoints(), fromFirewallMatch(), fromSomeMiddleAppMatch())

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "secure-intranet-connectivity",
			Labels: map[string]string{
				"app": "unknown-app",
			},
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 3)
		}),
	)

	_, err := server.Request(ctx, request)
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_MatchSelectedNSE(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	nsName := networkServiceName()
	nses := endpoints()

	nsServer, nseServer := testServers(t, nsName, nses, fromAnywhereMatch(), fromFirewallMatch(), fromSomeMiddleAppMatch())

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkServiceEndpointName: nses[0].GetName(),
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.NotNil(t, clienturlctx.ClientURL(ctx))
		}),
	)

	_, err := server.Request(ctx, request)
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_NoMatchServiceFound(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, "", endpoints())

	want := labels(nsName, map[string]string{
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
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].GetNetworkServiceLabels())
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	_, err := server.Request(ctx, request)
	require.Error(t, err)
}

func TestDiscoverCandidatesServer_NoMatchServiceEndpointFound(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	nsName := networkServiceName()

	nsServer, nseServer := testServers(t, nsName, []*registry.NetworkServiceEndpoint{}, fromFirewallMatch(), fromSomeMiddleAppMatch(), fromAnywhereMatch())

	want := labels(nsName, map[string]string{
		"app": "firewall",
	})

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: nsName,
			Labels:         map[string]string{},
		},
	}

	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, want, nses[0].GetNetworkServiceLabels())
		}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), testWait)
	defer cancel()

	_, err := server.Request(ctx, request)
	require.Error(t, err)
}

func TestDiscoverCandidatesServer_MatchExactService(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	nsServer, nseServer := testServers(t, "", []*registry.NetworkServiceEndpoint{})

	nsName := networkServiceName()
	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(nsServer),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			nses := discover.Candidates(ctx).Endpoints
			require.Len(t, nses, 1)
			require.Equal(t, nsName, nses[0].GetNetworkServiceNames()[0])
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
	ctx, cancel := context.WithTimeout(context.Background(), testWait)
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
		Name:    nsName,
		Payload: payload.IP,
	})
	require.NoError(t, err)
	_, err = nseServer.Register(context.Background(), &registry.NetworkServiceEndpoint{
		NetworkServiceNames: []string{nsName},
	})
	require.NoError(t, err)

	// 4. Try to discover NSE by the right NS name
	ctx, cancel = context.WithTimeout(context.Background(), testWait)
	defer cancel()

	conn, err := server.Request(ctx, request.Clone())
	require.NoError(t, err)
	require.Equal(t, payload.IP, conn.GetPayload())
}

func TestDiscoverCandidatesServer_MatchExactEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	nseServer := memory.NewNetworkServiceEndpointRegistryServer()

	nseName := "final-endpoint"
	u := "tcp://" + nseName
	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(memory.NewNetworkServiceRegistryServer()),
			registryadapters.NetworkServiceEndpointServerToClient(nseServer)),
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
	_, err = server.Request(ctx, request.Clone())
	require.NoError(t, err)
}

func TestDiscoverCandidatesServer_NoEndpointOnClose(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var closed bool
	server := next.NewNetworkServiceServer(
		discover.NewServer(
			registryadapters.NetworkServiceServerToClient(memory.NewNetworkServiceRegistryServer()),
			registryadapters.NetworkServiceEndpointServerToClient(memory.NewNetworkServiceEndpointRegistryServer())),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.Nil(t, clienturlctx.ClientURL(ctx))
			closed = true
		}),
	)

	conn := &networkservice.Connection{
		NetworkServiceEndpointName: "nse",
	}

	_, err := server.Close(ctx, conn)
	require.NoError(t, err)

	require.True(t, closed)
}

func Test_Discover_Scale_FromZero_vL3(t *testing.T) {
	ns := &registry.NetworkService{
		Name:    "ns-1",
		Payload: "IP",
		Matches: []*registry.Match{
			{
				Fallthrough: true,
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"app":      "nse-vl3-vpp",
							"nodeName": "{{.nodeName}}",
						},
					},
				},
			},
			{
				Fallthrough: true,
				SourceSelector: map[string]string{
					"capability": "vl3",
				},
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"capability": "vl3",
						},
					},
				},
			},
			{
				Fallthrough: true,
				Routes: []*registry.Destination{
					{
						DestinationSelector: map[string]string{
							"app": "nse-autoscaler",
						},
					},
				},
			},
		},
	}

	nseA := &registry.NetworkServiceEndpoint{
		Name: "nse-A",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"ns-1": {Labels: map[string]string{
				"capability": "vl3",
				"nodeName":   "node-1",
				"app":        "nse-vl3-vpp",
			}},
		},
	}

	nseB := &registry.NetworkServiceEndpoint{
		Name: "nse-B",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"ns-1": {Labels: map[string]string{
				"capability": "vl3",
				"nodeName":   "node-2",
				"app":        "nse-vl3-vpp",
			}},
		},
	}

	nseC := &registry.NetworkServiceEndpoint{
		Name: "nse-C",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"ns-1": {Labels: map[string]string{
				"app": "nse-autoscaler",
			}},
		},
	}

	nseD := &registry.NetworkServiceEndpoint{
		Name: "nse-D",
		NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
			"ns-1": {Labels: map[string]string{
				"app": "unknown",
			}},
		},
	}

	require.Empty(t, matchutils.MatchEndpoint(map[string]string{}, ns, nseD), "there are no matches for nse-D")
	require.Empty(t, matchutils.MatchEndpoint(map[string]string{"app": "nse-autoscaler"}, ns, nseD), "there are no matches for nse-D")

	require.Equal(t, []*registry.NetworkServiceEndpoint{nseC}, matchutils.MatchEndpoint(map[string]string{}, ns, nseC, nseD), "by match №3")
	require.Equal(t, []*registry.NetworkServiceEndpoint{nseC}, matchutils.MatchEndpoint(map[string]string{"app": "nse-autoscaler"}, ns, nseC, nseD), "by match №3")

	require.Equal(t, []*registry.NetworkServiceEndpoint{nseB}, matchutils.MatchEndpoint(map[string]string{"app": "nse-vl3-vpp", "nodeName": "node-2"}, ns, nseC, nseA, nseB, nseD), "by match №1")
	require.Equal(t, []*registry.NetworkServiceEndpoint{nseA}, matchutils.MatchEndpoint(map[string]string{"app": "nse-vl3-vpp", "nodeName": "node-1"}, ns, nseC, nseA, nseD, nseB), "by match №1")

	require.Equal(t, []*registry.NetworkServiceEndpoint{nseA}, matchutils.MatchEndpoint(map[string]string{"capability": "vl3", "app": "nse-vl3-vpp", "nodeName": "node-2"}, ns, nseC, nseA, nseD), "by match №2")
	require.Equal(t, []*registry.NetworkServiceEndpoint{nseB}, matchutils.MatchEndpoint(map[string]string{"capability": "vl3", "app": "nse-vl3-vpp", "nodeName": "node-1"}, ns, nseC, nseD, nseB), "by match №2")
}
