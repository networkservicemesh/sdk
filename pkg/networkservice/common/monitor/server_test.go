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

package monitor_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
)

type dummyMonitorServer struct {
	networkservice.MonitorConnectionServer
}

type updateConnServer struct {
	requestFunc func(ctx context.Context, request *networkservice.NetworkServiceRequest) *networkservice.Connection
}

func newUpdateConnServer(requestFunc func(ctx context.Context, request *networkservice.NetworkServiceRequest) *networkservice.Connection) *updateConnServer {
	return &updateConnServer{
		requestFunc: requestFunc,
	}
}

func (t *updateConnServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return t.requestFunc(ctx, request), nil
}

func (t *updateConnServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func TestMonitorSendToRightClient(t *testing.T) {
	myServer := &dummyMonitorServer{}

	ms := monitor.NewServer(&myServer.MonitorConnectionServer)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := newTestMonitorClient(ctx, myServer, "local-nsm")
	remoteMonitor := newTestMonitorClient(ctx, myServer, "remote-nsm")

	// We need to be sure we have 2 clients waiting for Events, we could check to have initial transfers for this.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	remoteMonitor.WaitEvents(timeoutCtx, 1)
	// Check we have 2 monitors
	require.Equal(t, 1, len(localMonitor.Events))
	require.Equal(t, 1, len(remoteMonitor.Events))

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", "local-nsm"),
	}
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.
	// Wait for update message for first monitor
	localMonitor.WaitEvents(timeoutCtx, 2)
	require.Equal(t, len(localMonitor.Events), 2)
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.Events[1].Type)

	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	conn2, _ := srv.Request(ctx, nsr)
	require.NotNil(t, conn2)
	localMonitor.WaitEvents(timeoutCtx, 3)
	require.Equal(t, len(localMonitor.Events), 3)
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.Events[2].Type)
	// check delete event is working fine.
	_, closeErr := srv.Close(ctx, nsr.Connection)
	require.Nil(t, closeErr)
	localMonitor.WaitEvents(timeoutCtx, 4)
	// Last event should be delete
	require.Equal(t, 4, len(localMonitor.Events))
	require.Equal(t, networkservice.ConnectionEventType_DELETE, localMonitor.Events[3].Type)
}

func newUpdateTailServer(updateCounter int) *updateConnServer {
	updateEnv := newUpdateConnServer(func(ctx context.Context, request *networkservice.NetworkServiceRequest) *networkservice.Connection {
		updateCounter++
		connection := request.GetConnection()
		if connection.Labels == nil {
			connection.Labels = make(map[string]string)
		}
		connection.Labels["lastOp"] = fmt.Sprintf("updates: %v time: %v", updateCounter, time.Now())

		return request.GetConnection()
	})
	return updateEnv
}

func TestMonitorIsClosedProperly(t *testing.T) {
	myServer := &dummyMonitorServer{}

	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()
	ms := monitor.NewServer(&myServer.MonitorConnectionServer)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := newTestMonitorClient(ctx, myServer, "local-nsm")

	// We need to be sure we have 2 clients waiting for Events, we could check to have initial transfers for this.

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*6000)
	defer cancel()

	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	require.Equal(t, 1, len(localMonitor.Events))

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", "local-nsm"),
	}
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.

	// Wait for update message for first monitor
	localMonitor.WaitEvents(timeoutCtx, 2)

	require.Equal(t, 2, len(localMonitor.Events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.Events[1].Type)

	// Cancel context for monitor
	cancel()

	ctx, cancelFunc = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	newLocalMon := newTestMonitorClient(ctx, myServer, "local-nsm")

	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)

	newLocalMon.WaitEvents(timeoutCtx, 2)
}

func createConnection(id, server string) *networkservice.Connection {
	return &networkservice.Connection{
		Id: id,
		Path: &networkservice.Path{
			Index: 0,
			PathSegments: []*networkservice.PathSegment{
				{
					Name: server,
				},
			},
		},
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpRequired: true,
				DstIpRequired: true,
			},
		},
	}
}

// testMonitorClient - implementation of test monitor client.
type testMonitorClient struct {
	Events       []*networkservice.ConnectionEvent
	eventChannel chan *networkservice.ConnectionEvent
	ctx          context.Context
	grpc.ServerStream
	executor  serialize.Executor
	finalized chan struct{}
}

// newTestMonitorClient - construct a new client.
func newTestMonitorClient(ctx context.Context, server networkservice.MonitorConnectionServer, segmentName string) *testMonitorClient {
	rv := &testMonitorClient{
		eventChannel: make(chan *networkservice.ConnectionEvent, 10),
		ctx:          ctx,
		executor:     serialize.NewExecutor(),
		finalized:    make(chan struct{}),
	}
	runtime.SetFinalizer(rv, func(server *testMonitorClient) {
		close(server.finalized)
	})

	go func() {
		select {
		case <-ctx.Done():
		default:
			_ = server.MonitorConnections(
				&networkservice.MonitorScopeSelector{
					PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
				}, rv)
		}
	}()

	return rv
}

// Send - receive event from server.
func (t *testMonitorClient) Send(evt *networkservice.ConnectionEvent) error {
	t.executor.SyncExec(func() {
		t.Events = append(t.Events, evt)
		t.eventChannel <- evt
	})
	return nil
}

// Context - current context to perform checks.
func (t *testMonitorClient) Context() context.Context {
	return t.ctx
}

// WaitEvents - wait for a required number of events to be received.
func (t *testMonitorClient) WaitEvents(ctx context.Context, count int) {
	for {
		var curLen = 0
		t.executor.SyncExec(func() {
			curLen = len(t.Events)
		})
		if curLen >= count {
			logrus.Infof("Waiting for Events %v Complete", count)
			break
		}
		// Wait for event to occur or timeout happen.
		select {
		case <-ctx.Done():
			// Context is done, we need to exit
			logrus.Errorf("Failed to wait for Events count %v current value is: %v", count, curLen)
			return
		case <-t.eventChannel:
		}
	}
}
