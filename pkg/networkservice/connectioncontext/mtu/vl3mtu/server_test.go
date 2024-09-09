// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package vl3mtu_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/connectioncontext/mtu/vl3mtu"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

func Test_vl3MtuServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Specify pathSegments to test
	segmentNames := []string{"local-nsm", "remote-nsm"}

	// Create monitorServer
	var monitorServer networkservice.MonitorConnectionServer
	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		monitor.NewServer(ctx, &monitorServer),
		vl3mtu.NewServer(),
	)
	monitorClient := adapters.NewMonitorServerToClient(monitorServer)

	// Create maps to hold returned connections and receivers
	connections := make(map[string]*networkservice.Connection)
	receivers := make(map[string]networkservice.MonitorConnection_MonitorConnectionsClient)

	// Get Empty initial state transfers
	for _, segmentName := range segmentNames {
		monitorCtx, cancelMonitor := context.WithCancel(ctx)
		defer cancelMonitor()

		var monitorErr error
		receivers[segmentName], monitorErr = monitorClient.MonitorConnections(monitorCtx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
		})
		require.NoError(t, monitorErr)

		event, err := receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())
		require.Empty(t, event.GetConnections()[segmentName].GetPath().GetPathSegments())
	}

	// Send requests
	var err error
	connections[segmentNames[0]], err = server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: segmentNames[0],
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: []*networkservice.PathSegment{{Name: segmentNames[0]}},
			},
		},
	})
	require.NoError(t, err)

	// Send requests with different mtu
	connections[segmentNames[1]], err = server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: segmentNames[1],
			Path: &networkservice.Path{
				Index:        1,
				PathSegments: []*networkservice.PathSegment{{Name: segmentNames[1]}},
			},
			Context: &networkservice.ConnectionContext{MTU: 1500},
		},
	})
	require.NoError(t, err)

	// Get Updates and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		var event *networkservice.ConnectionEvent
		event, err = receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
		require.Len(t, event.GetConnections()[segmentName].GetPath().GetPathSegments(), 1)
		require.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}

	// The first client should receive REFRESH_REQUESTED state, because MTU was updated
	event, err := receivers[segmentNames[0]].Recv()
	require.NoError(t, err)

	require.NotNil(t, event)
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
	require.Equal(t, networkservice.State_REFRESH_REQUESTED, event.GetConnections()[segmentNames[0]].GetState())
	require.Len(t, event.GetConnections()[segmentNames[0]].GetPath().GetPathSegments(), 1)
	require.Equal(t, segmentNames[0], event.GetConnections()[segmentNames[0]].GetPath().GetPathSegments()[0].GetName())

	// Close Connections
	for _, conn := range connections {
		_, err := server.Close(ctx, conn)
		require.NoError(t, err)
	}

	// Get deleteMonitorClientCC Events and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		event, err := receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_DELETE, event.GetType())
		require.Len(t, event.GetConnections()[segmentName].GetPath().GetPathSegments(), 1)
		require.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}
}

func Test_vl3MtuServer_SpoiledConnection(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	const (
		id1       = "id1"
		id2       = "id2"
		invalidID = "invalidId"
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	segmentName := "local-nsm"

	// Create monitorServer
	var monitorServer networkservice.MonitorConnectionServer
	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		monitor.NewServer(ctx, &monitorServer),
		vl3mtu.NewServer(),
	)
	monitorClient := adapters.NewMonitorServerToClient(monitorServer)
	monitorCtx, cancelMonitor := context.WithCancel(ctx)
	defer cancelMonitor()

	receiver, monitorErr := monitorClient.MonitorConnections(monitorCtx, &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
	})
	require.NoError(t, monitorErr)

	// Get Empty initial state transfer
	event, err := receiver.Recv()
	require.NoError(t, err)
	require.NotNil(t, event)
	require.Equal(t, networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())

	// Send the first request
	connection1, err := server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: id1,
			Path: &networkservice.Path{
				PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
			},
		},
	})
	require.NoError(t, err)

	// Get an update that was triggered by the first request and insure we've properly filtered by segmentName
	_, err = receiver.Recv()
	require.NoError(t, err)

	// Spoil the connectionId of the first connection
	connection1.Id = invalidID

	// Send the second request
	_, err = server.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: id2,
			Path: &networkservice.Path{
				PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
			},
			Context: &networkservice.ConnectionContext{MTU: 1500},
		},
	})
	require.NoError(t, err)

	// Get an update that was triggered by the second request and insure we've properly filtered by segmentName
	_, err = receiver.Recv()
	require.NoError(t, err)

	// We should get the event that contains only the proper connection
	event, err = receiver.Recv()
	require.NotNil(t, event)
	require.NoError(t, err)
	eventConn, ok := event.GetConnections()[id1]
	require.True(t, ok)
	require.Equal(t, networkservice.State_REFRESH_REQUESTED, eventConn.GetState())

	_, ok = event.GetConnections()[invalidID]
	require.False(t, ok)
}
