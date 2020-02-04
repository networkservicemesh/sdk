package monitor

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
)

type testMonitorServer struct {
	events       []*networkservice.ConnectionEvent
	eventChannel chan *networkservice.ConnectionEvent
	ctx          context.Context
	cancel       context.CancelFunc
}

func newTestMonitorServer() *testMonitorServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &testMonitorServer{
		eventChannel: make(chan *networkservice.ConnectionEvent, 10),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (t *testMonitorServer) Send(evt *networkservice.ConnectionEvent) error {
	t.events = append(t.events, evt)
	t.eventChannel <- evt
	return nil
}

func (t *testMonitorServer) SetHeader(metadata.MD) error {
	return nil
}

func (t *testMonitorServer) SendHeader(metadata.MD) error {
	return nil
}

func (t *testMonitorServer) SetTrailer(metadata.MD) {
}

func (t *testMonitorServer) Context() context.Context {
	return t.ctx
}

func (t *testMonitorServer) SendMsg(m interface{}) error {
	return nil
}

func (t *testMonitorServer) RecvMsg(m interface{}) error {
	return nil
}

func (t *testMonitorServer) BeginMonitoring(server networkservice.MonitorConnectionServer, segmentName string) {
	go func() {
		_ = server.MonitorConnections(
			&networkservice.MonitorScopeSelector{
				PathSegments: []*networkservice.PathSegment{{Name: segmentName}},
			}, t)
	}()
}

func (t *testMonitorServer) WaitEvents(ctx context.Context, count int) {
	for {
		if len(t.events) == count {
			logrus.Infof("Waiting for events %v, but has %v", count, len(t.events))
			break
		}
		// Wait 10ms for listeners to activate
		select {
		case <-ctx.Done():
			// Context is done, we need to exit
			logrus.Errorf("Failed to wait for events count %v current value is: %v", count, len(t.events))
			return
		case <-t.eventChannel:
		case <-time.After(1 * time.Second):
		}
	}
}

type myServer struct {
	networkservice.MonitorConnectionServer
}

type updateConnServer struct {
	requestFunc func(request *networkservice.NetworkServiceRequest) *networkservice.Connection
}

func newUpdateConnServer(requestFunc func(request *networkservice.NetworkServiceRequest) *networkservice.Connection) *updateConnServer {
	return &updateConnServer{
		requestFunc: requestFunc,
	}
}

func (t *updateConnServer) Request(_ context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return t.requestFunc(request), nil
}

func (t *updateConnServer) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func TestMonitorSendToRightClient(t *testing.T) {
	myServer := &myServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := newTestMonitorServer()
	remoteMonitor := newTestMonitorServer()
	localMonitor.BeginMonitoring(myServer, "local-nsm")
	remoteMonitor.BeginMonitoring(myServer, "remote-nsm")
	// We need to be sure we have 2 clients waiting for events, we could check to have initial transfers for this.
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*6000)
	defer cancel()
	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	remoteMonitor.WaitEvents(timeoutCtx, 1)
	// Check we have 2 monitors
	require.Equal(t, len(mons.monitors), 2)
	require.Equal(t, 1, len(localMonitor.events))
	require.Equal(t, 1, len(remoteMonitor.events))

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", "local-nsm"),
	}
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.
	// Wait for update message for first monitor
	localMonitor.WaitEvents(timeoutCtx, 2)
	require.Equal(t, 2, len(localMonitor.events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.events[1].Type)
	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)
	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)
	localMonitor.WaitEvents(timeoutCtx, 3)
	require.Equal(t, 3, len(localMonitor.events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.events[2].Type)
	// check delete event is working fine.
	_, closeErr := srv.Close(ctx, nsr.Connection)
	require.Nil(t, closeErr)
	localMonitor.WaitEvents(timeoutCtx, 4)
	// Last event should be delete
	require.Equal(t, 4, len(localMonitor.events))
	require.Equal(t, networkservice.ConnectionEventType_DELETE, localMonitor.events[3].Type)
	// Check connection is not longer inside map
	require.Equal(t, len(mons.connections), 0)
}

func newUpdateTailServer(updateCounter int) *updateConnServer {
	updateEnv := newUpdateConnServer(func(request *networkservice.NetworkServiceRequest) *networkservice.Connection {
		updateCounter++
		if request.GetConnection().Labels == nil {
			request.GetConnection().Labels = make(map[string]string)
		}
		request.GetConnection().Labels["lastOp"] = fmt.Sprintf("updates: %v time: %v", updateCounter, time.Now())
		return request.GetConnection()
	})
	return updateEnv
}

func TestMonitorIsClosedProperly(t *testing.T) {
	myServer := &myServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	updateCounter := 0
	updateEnv := newUpdateTailServer(updateCounter)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, updateEnv)

	localMonitor := newTestMonitorServer()

	localMonitor.BeginMonitoring(myServer, "local-nsm")

	// We need to be sure we have 2 clients waiting for events, we could check to have initial transfers for this.

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*6000)
	defer cancel()

	// Wait for init messages in both monitors
	localMonitor.WaitEvents(timeoutCtx, 1)
	require.Equal(t, 1, len(localMonitor.events))

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", "local-nsm"),
	}
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.

	// Wait for update message for first monitor
	localMonitor.WaitEvents(timeoutCtx, 2)

	require.Equal(t, 2, len(localMonitor.events))
	require.Equal(t, networkservice.ConnectionEventType_UPDATE, localMonitor.events[1].Type)

	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)

	// Cancel context for monitor
	localMonitor.cancel()
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
