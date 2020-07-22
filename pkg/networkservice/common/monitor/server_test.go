// Copyright (c) 2020 Cisco and/or its affiliates.
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

package monitor_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
)

func TestMonitor(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	// Specify pathSegments to test
	segmentNames := []string{"local-nsm", "remote-nsm"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Create monitorServer, monitorClient, and server.
	var monitorServer networkservice.MonitorConnectionServer
	server := monitor.NewServer(ctx, &monitorServer)
	monitorClient := adapters.NewMonitorServerToClient(monitorServer)

	// Create maps to hold returned connections and receivers
	connections := make(map[string]*networkservice.Connection)
	receivers := make(map[string]networkservice.MonitorConnection_MonitorConnectionsClient)

	// Get Empty initial state transfers
	for _, segmentName := range segmentNames {
		var err error
		ctx, cancelFunc := context.WithCancel(context.Background())
		defer cancelFunc()
		receivers[segmentName], err = monitorClient.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
		})
		assert.Nil(t, err)
		event, err := receivers[segmentName].Recv()
		assert.Nil(t, err)
		require.NotNil(t, event)
		assert.Equal(t, networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())
		require.Equal(t, len(event.GetConnections()[segmentName].GetPath().GetPathSegments()), 0)
	}

	// Send requests
	for _, segmentName := range segmentNames {
		var err error
		connections[segmentName], err = server.Request(context.Background(), &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: segmentName,
				Path: &networkservice.Path{
					Index: 0,
					PathSegments: []*networkservice.PathSegment{
						{
							Name: segmentName,
						},
					},
				},
			},
		})
		assert.Nil(t, err)
	}

	// Get Updates and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		event, err := receivers[segmentName].Recv()
		assert.Nil(t, err)
		require.NotNil(t, event)
		assert.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
		require.Equal(t, len(event.GetConnections()[segmentName].GetPath().GetPathSegments()), 1)
		assert.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}

	// Close Connections
	for _, conn := range connections {
		_, err := server.Close(context.Background(), conn)
		assert.Nil(t, err)
	}

	// Get Delete Events and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		event, err := receivers[segmentName].Recv()
		assert.Nil(t, err)
		require.NotNil(t, event)
		assert.Equal(t, networkservice.ConnectionEventType_DELETE, event.GetType())
		require.Equal(t, len(event.GetConnections()[segmentName].GetPath().GetPathSegments()), 1)
		assert.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}
}
