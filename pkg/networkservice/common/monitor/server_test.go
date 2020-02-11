package monitor

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/test/test_monitor"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
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

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.True(t, ok)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := test_monitor.NewTestMonitorClient()
	remoteMonitor := test_monitor.NewTestMonitorClient()
	localMonitor.BeginMonitoring(myServer, "local-nsm")
	remoteMonitor.BeginMonitoring(myServer, "remote-nsm")
	// We need to be sure we have 2 clients waiting for Events, we could check to have initial transfers for this.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	remoteMonitor.WaitEvents(timeoutCtx, 1)
	// Check we have 2 monitors
	require.Equal(t, len(mons.monitors), 2)
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
	require.Equal(t, 2, len(localMonitor.Events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.Events[1].Type)
	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)
	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)
	localMonitor.WaitEvents(timeoutCtx, 3)
	require.Equal(t, 3, len(localMonitor.Events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.Events[2].Type)
	// check delete event is working fine.
	_, closeErr := srv.Close(ctx, nsr.Connection)
	require.Nil(t, closeErr)
	localMonitor.WaitEvents(timeoutCtx, 4)
	// Last event should be delete
	require.Equal(t, 4, len(localMonitor.Events))
	require.Equal(t, networkservice.ConnectionEventType_DELETE, localMonitor.Events[3].Type)
	// Check connection is not longer inside map
	require.Equal(t, len(mons.connections), 0)
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

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := test_monitor.NewTestMonitorClient()

	localMonitor.BeginMonitoring(myServer, "local-nsm")

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

	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)

	// Cancel context for monitor
	localMonitor.Cancel()
	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)

	for {
		if len(mons.monitors) == 0 {
			logrus.Infof("Waiting for monitors %v, but has %v", 0, len(mons.monitors))
			break
		}
		// Wait 10ms for listeners to activate
		select {
		case <-ctx.Done():
			// Context is done, we need to exit
			require.Fail(t, "Context timeout")
		case <-time.After(time.Millisecond * 100): // Delay for channel to be activated
		}
	}
	// Check we no have monitors anymore
	require.Equal(t, len(mons.monitors), 0)
}

func TestChainedIntervalSend(t *testing.T) {
	myServer := &dummyMonitorServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := test_monitor.NewTestMonitorClient()
	localMonitor.BeginMonitoring(myServer, "local-nsm")
	// We need to be sure we have 2 clients waiting for Events, we could check to have initial transfers for this.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*6000)
	defer cancel()
	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	// Check we have 2 monitors
	require.Equal(t, len(mons.monitors), 1)
	require.Equal(t, 1, len(localMonitor.Events))

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", "local-nsm"),
	}

	conn, _ := srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.
	// Wait for update message for first monitor
	localMonitor.WaitEvents(timeoutCtx, 2)

	mons.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			conn.GetId(): createMetrics(conn, "mx1"),
		},
	})
	mons.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			conn.GetId(): createMetrics(conn, "mx2"),
		},
	})
	mons.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			conn.GetId(): createMetrics(conn, "mx3"),
		},
	})
	mons.executor.SyncExec(func() { /* Dummy */ })

	localMonitor.WaitEvents(timeoutCtx, 3)
	// Check we still have 3 Events, first since never send, and 2 metrics updates are in pending state.
	require.Equal(t, 3, len(localMonitor.Events))

	require.Equal(t, "mx1", localMonitor.Events[2].Connections[conn.GetId()].GetPath().GetPathSegments()[0].Metrics["mx:"])

	// Check pending update is mx3
	require.Equal(t, "mx3", mons.connections[conn.GetId()].connection.GetPath().GetPathSegments()[0].Metrics["mx:"])
	require.Equal(t, 2, mons.connections[conn.GetId()].pendingUpdates)

	conn, _ = srv.Request(ctx, nsr)
	localMonitor.WaitEvents(timeoutCtx, 4)
	require.Equal(t, "mx3", localMonitor.Events[3].Connections[conn.GetId()].GetPath().GetPathSegments()[0].Metrics["mx:"])
}

func createMetrics(conn *networkservice.Connection, mx string) *networkservice.Connection {
	cpy := proto.Clone(conn).(*networkservice.Connection)
	for pos, s := range cpy.GetPath().GetPathSegments() {
		s.Metrics = map[string]string{
			"mx:": mx,
			"pos": fmt.Sprintf("%v", pos),
			"dt":  fmt.Sprintf("%v %v", conn.GetPath().GetIndex(), time.Now()),
		}
	}
	return cpy
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
			MetricsContext: &networkservice.MetricsContext{
				Enabled:  true,
				Interval: int64(1 * time.Hour),
			},
		},
	}
}
