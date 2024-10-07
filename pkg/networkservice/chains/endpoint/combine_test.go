// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2024 Cisco and/or its affiliates.
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

package endpoint_test

import (
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/memif"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startEndpoint(ctx context.Context, t *testing.T, e endpoint.Endpoint) *grpc.ClientConn {
	listenOn := &url.URL{Scheme: "tcp", Path: "127.0.0.1:0"}

	require.Empty(t, endpoint.Serve(ctx, listenOn, e))

	cc, err := grpc.Dial(grpcutils.URLToTarget(listenOn), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	return cc
}

func TestCombine(t *testing.T) {
	var samples = []struct {
		name      string
		mechanism *networkservice.Mechanism
	}{
		{
			name:      "Kernel",
			mechanism: kernel.New(""),
		},
		{
			name:      "Memif",
			mechanism: memif.New(""),
		},
	}

	for _, s := range samples {
		t.Run(s.name, func(t *testing.T) {
			testCombine(t, s.mechanism)
		})
	}
}

func testCombine(t *testing.T, mechanism *networkservice.Mechanism) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	e := endpoint.Combine(func(servers []networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
		return mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernel.MECHANISM: servers[0],
			memif.MECHANISM:  servers[1],
		})
	}, newTestEndpoint(ctx, kernel.MECHANISM, kernel.MECHANISM), newTestEndpoint(ctx, memif.MECHANISM, memif.MECHANISM))

	cc := startEndpoint(ctx, t, e)
	defer func() { _ = cc.Close() }()

	c := next.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		networkservice.NewNetworkServiceClient(cc),
	)

	stream, err := networkservice.NewMonitorConnectionClient(cc).MonitorConnections(ctx, &networkservice.MonitorScopeSelector{})
	require.NoError(t, err)

	// 1. Receive initial event
	event, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, (&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
	}).String(), event.String())

	// 2. Request and receive UPDATE event
	conn, err := c.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection:           new(networkservice.Connection),
		MechanismPreferences: []*networkservice.Mechanism{mechanism.Clone()},
	})
	require.NoError(t, err)
	require.Equal(t, mechanism.String(), conn.GetMechanism().String())
	require.Len(t, conn.GetPath().GetPathSegments(), 2)
	require.Equal(t, mechanism.Type, conn.GetNextPathSegment().GetName())

	event, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, (&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			conn.GetCurrentPathSegment().GetId(): {
				Id:        conn.GetCurrentPathSegment().GetId(),
				Mechanism: conn.GetMechanism(),
				Path: &networkservice.Path{
					Index:        0,
					PathSegments: conn.GetPath().GetPathSegments(),
				},
			},
		},
	}).String(), event.String())

	// 3. Close and receive DELETE event
	_, err = c.Close(ctx, conn.Clone())
	require.NoError(t, err)

	event, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, (&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{
			conn.GetNextPathSegment().GetId(): {
				Id:        conn.GetNextPathSegment().GetId(),
				Mechanism: conn.GetMechanism(),
				Path: &networkservice.Path{
					Index:        1,
					PathSegments: conn.GetPath().GetPathSegments(),
				},
			},
		},
	}).String(), event.String())
}

func TestSwitchEndpoint_InitialStateTransfer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	e := endpoint.Combine(func(servers []networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
		return mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernel.MECHANISM: servers[0],
			memif.MECHANISM:  servers[1],
		})
	}, newTestEndpoint(ctx, kernel.MECHANISM, kernel.MECHANISM), newTestEndpoint(ctx, memif.MECHANISM, memif.MECHANISM))

	cc := startEndpoint(ctx, t, e)
	defer func() { _ = cc.Close() }()

	c := next.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		networkservice.NewNetworkServiceClient(cc),
	)

	var conns []*networkservice.Connection
	for _, mechanism := range []*networkservice.Mechanism{kernel.New(""), memif.New("")} {
		conn, err := c.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection:           new(networkservice.Connection),
			MechanismPreferences: []*networkservice.Mechanism{mechanism.Clone()},
		})
		require.NoError(t, err)
		require.Equal(t, mechanism.String(), conn.GetMechanism().String())
		require.Len(t, conn.GetPath().GetPathSegments(), 2)
		require.Equal(t, mechanism.Type, conn.GetNextPathSegment().GetName())

		conns = append(conns, conn)
	}

	expectedEvent := &networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: make(map[string]*networkservice.Connection),
	}
	for _, conn := range conns {
		expectedEvent.Connections[conn.GetCurrentPathSegment().GetId()] = &networkservice.Connection{
			Id:        conn.GetCurrentPathSegment().GetId(),
			Mechanism: conn.GetMechanism(),
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: conn.GetPath().GetPathSegments(),
			},
		}
	}

	ignoreCurrent := goleak.IgnoreCurrent()
	streamCtx, cancelStream := context.WithCancel(ctx)

	stream, err := networkservice.NewMonitorConnectionClient(cc).MonitorConnections(streamCtx, &networkservice.MonitorScopeSelector{})
	require.NoError(t, err)

	event, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, expectedEvent.String(), event.String())

	cancelStream()
	goleak.VerifyNone(t, ignoreCurrent)
}

func TestSwitchEndpoint_DuplicateEndpoints(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	monitorCtx, cancelMonitor := context.WithCancel(ctx)

	duplicate1 := newTestEndpoint(monitorCtx, kernel.MECHANISM, "duplicate")
	duplicate2 := newTestEndpoint(monitorCtx, memif.MECHANISM, "duplicate")
	e := endpoint.Combine(func(servers []networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
		return mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernel.MECHANISM: servers[0],
			memif.MECHANISM:  servers[1],
		})
	}, duplicate1, duplicate2)

	cc := startEndpoint(ctx, t, e)
	defer func() { _ = cc.Close() }()

	c := next.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		networkservice.NewNetworkServiceClient(cc),
	)

	stream, err := networkservice.NewMonitorConnectionClient(cc).MonitorConnections(ctx, &networkservice.MonitorScopeSelector{})
	require.NoError(t, err)

	event, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())

	var conns []*networkservice.Connection
	for _, mechanism := range []*networkservice.Mechanism{kernel.New(""), memif.New("")} {
		var conn *networkservice.Connection
		conn, err = c.Request(ctx, &networkservice.NetworkServiceRequest{
			Connection:           new(networkservice.Connection),
			MechanismPreferences: []*networkservice.Mechanism{mechanism.Clone()},
		})
		require.NoError(t, err)

		conns = append(conns, conn)

		event, err = stream.Recv()
		require.NoError(t, err)
		require.Equal(t, (&networkservice.ConnectionEvent{
			Type: networkservice.ConnectionEventType_UPDATE,
			Connections: map[string]*networkservice.Connection{
				conn.GetCurrentPathSegment().GetId(): {
					Id:        conn.GetCurrentPathSegment().GetId(),
					Mechanism: conn.GetMechanism(),
					Path: &networkservice.Path{
						Index:        0,
						PathSegments: conn.GetPath().GetPathSegments(),
					},
				},
			},
		}).String(), event.String())
	}

	for _, conn := range conns {
		_, err = c.Close(ctx, conn.Clone())
		require.NoError(t, err)

		event, err = stream.Recv()
		require.NoError(t, err)
		require.Equal(t, (&networkservice.ConnectionEvent{
			Type: networkservice.ConnectionEventType_DELETE,
			Connections: map[string]*networkservice.Connection{
				conn.GetNextPathSegment().GetId(): {
					Id:        conn.GetNextPathSegment().GetId(),
					Mechanism: conn.GetMechanism(),
					Path: &networkservice.Path{
						Index:        1,
						PathSegments: conn.GetPath().GetPathSegments(),
					},
				},
			},
		}).String(), event.String())
	}

	cancelMonitor()

	_, err = stream.Recv()
	require.ErrorIs(t, err, io.EOF)
}

type testEndpoint struct {
	networkservice.NetworkServiceServer
	networkservice.MonitorConnectionServer
}

func newTestEndpoint(ctx context.Context, mechanism, name string) *testEndpoint {
	e := new(testEndpoint)
	e.NetworkServiceServer = next.NewNetworkServiceServer(
		begin.NewServer(),
		metadata.NewServer(),
		monitor.NewServer(ctx, &e.MonitorConnectionServer),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			mechanism: updatepath.NewServer(name),
		}),
	)
	return e
}

func (e *testEndpoint) Register(s *grpc.Server) {
	grpcutils.RegisterHealthServices(s, e)
	networkservice.RegisterNetworkServiceServer(s, e)
	networkservice.RegisterMonitorConnectionServer(s, e)
}
