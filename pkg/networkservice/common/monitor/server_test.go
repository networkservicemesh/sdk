// Copyright (c) 2020-2024 Cisco and/or its affiliates.
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
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
)

func TestMonitorServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Specify pathSegments to test
	segmentNames := []string{"local-nsm", "remote-nsm"}

	// Create monitorServer, monitorClient, and server.
	var monitorServer networkservice.MonitorConnectionServer
	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		monitor.NewServer(ctx, &monitorServer),
	)
	monitorClient := adapters.NewMonitorServerToClient(monitorServer)

	// Create maps to hold returned connections and receivers
	connections := make(map[string]*networkservice.Connection)
	receivers := make(map[string]networkservice.MonitorConnection_MonitorConnectionsClient)

	// Create non-reading monitor client for all connections
	_, monitorErr := monitorClient.MonitorConnections(ctx, new(networkservice.MonitorScopeSelector))
	require.NoError(t, monitorErr)

	// Get Empty initial state transfers
	for _, segmentName := range segmentNames {
		monitorCtx, cancelMonitor := context.WithCancel(ctx)
		defer cancelMonitor()

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
	for _, segmentName := range segmentNames {
		var err error
		connections[segmentName], err = server.Request(ctx, &networkservice.NetworkServiceRequest{
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
		require.NoError(t, err)
	}

	// Get Updates and insure we've properly filtered by segmentName
	for _, segmentName := range segmentNames {
		event, err := receivers[segmentName].Recv()
		require.NoError(t, err)

		require.NotNil(t, event)
		require.Equal(t, networkservice.ConnectionEventType_UPDATE, event.GetType())
		require.Len(t, event.GetConnections()[segmentName].GetPath().GetPathSegments(), 1)
		require.Equal(t, segmentName, event.GetConnections()[segmentName].GetPath().GetPathSegments()[0].GetName())
	}

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

func TestMonitorServer_RequestConnEqualsToMonitorConn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	domain := sandbox.NewBuilder(ctx, t).
		SetNodesCount(1).
		Build()

	// Create forwarder that adds metrics to the connection
	for _, forwarder := range domain.Nodes[0].Forwarders {
		forwarder.Cancel()
	}
	domain.Nodes[0].NewForwarder(ctx, &registry.NetworkServiceEndpoint{
		Name:                sandbox.UniqueName("forwarder-metrics"),
		NetworkServiceNames: []string{"forwarder"},
	}, sandbox.GenerateTestToken, sandbox.WithForwarderAdditionalFunctionalityServer(&metricsServer{}))

	// Create NSE
	nsRegistryClient := domain.NewNSRegistryClient(ctx, sandbox.GenerateTestToken)
	nsReg := &registry.NetworkService{Name: "my-service"}
	_, err := nsRegistryClient.Register(ctx, nsReg)
	require.NoError(t, err)

	nseReg := &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{nsReg.Name},
	}
	domain.Nodes[0].NewEndpoint(ctx, nseReg, sandbox.GenerateTestToken)

	// Send Request
	connID := "1"
	nsc := domain.Nodes[0].NewClient(ctx, sandbox.GenerateTestToken)
	req := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernelmech.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             connID,
			NetworkService: nsReg.Name,
		},
	}

	requestConn, err := nsc.Request(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, requestConn)

	// Connect to NSMgr Monitor server to get actual connections
	target := grpcutils.URLToTarget(domain.Nodes[0].NSMgr.URL)
	cc, err := grpc.DialContext(ctx, target, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	require.NotNil(t, cc)
	go func(ctx context.Context, cc *grpc.ClientConn) {
		<-ctx.Done()
		_ = cc.Close()
	}(ctx, cc)
	c := networkservice.NewMonitorConnectionClient(cc)
	mc, err := c.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{})
	require.NoError(t, err)

	// eventConn must be equal to connection requestConn
	monitorEvent, _ := mc.Recv()
	for _, eventConn := range monitorEvent.GetConnections() {
		eventConn.Path.Index = 0
		eventConn.Id = connID
		require.True(t, requestConn.Equals(eventConn))
	}

	_, err = nsc.Close(ctx, requestConn)
	require.NoError(t, err)
}

func TestMonitorServer_FailedConnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Create a grpc connection to non existing address
	cc, err := grpc.Dial("1.1.1.1:5000", grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	require.NotNil(t, cc)

	// Create a server
	var monitorServer networkservice.MonitorConnectionServer
	server := chain.NewNetworkServiceServer(
		metadata.NewServer(),
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			clientconn.Store(ctx, cc)
		}),
		monitor.NewServer(ctx, &monitorServer),
	)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{Id: "id"},
	}

	// Make a request that should be successful and immediate (because monitor server connects to 1.1.1.1:5000 in background)
	conn, err := server.Request(ctx, request)
	require.NoError(t, err)
	require.NotNil(t, conn)
}

type metricsServer struct{}

func (m *metricsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	c, err := next.Server(ctx).Request(ctx, request)
	c.GetPath().GetPathSegments()[c.GetPath().GetIndex()].Metrics = map[string]string{"metricsServer": "1"}
	return c, err
}

func (m *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
