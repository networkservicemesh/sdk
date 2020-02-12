package monitor

import (
	"context"
	"fmt"
	"testing"
	"time"

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

	ctx := context.Background()
	ms := monitor.NewServer(&myServer.MonitorConnectionServer)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := NewTestMonitorClient()
	remoteMonitor := NewTestMonitorClient()
	localMonitor.BeginMonitoring(myServer, "local-nsm")
	remoteMonitor.BeginMonitoring(myServer, "remote-nsm")
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

	ctx := context.Background()
	ms := monitor.NewServer(&myServer.MonitorConnectionServer)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := NewTestMonitorClient()

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

	// Cancel context for monitor
	localMonitor.Cancel()

	newLocalMon := NewTestMonitorClient()
	newLocalMon.BeginMonitoring(myServer, "local-nsm")

	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)

	localMonitor.WaitEvents(timeoutCtx, 2)
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
