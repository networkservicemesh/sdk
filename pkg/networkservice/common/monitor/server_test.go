package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/connection"
	"github.com/networkservicemesh/api/pkg/api/connectioncontext"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
)

type testMonitorServer struct {
	events       []*connection.ConnectionEvent
	eventChannel chan *connection.ConnectionEvent
	ctx          context.Context
	cancel       context.CancelFunc
}

func newTestMonitorServer() *testMonitorServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &testMonitorServer{
		eventChannel: make(chan *connection.ConnectionEvent, 10),
		ctx:          ctx,
		cancel:       cancel,
	}
}

func (t *testMonitorServer) Send(evt *connection.ConnectionEvent) error {
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

func (t *testMonitorServer) BeginMonitoring(server connection.MonitorConnectionServer, segmentName string) {
	go func() {
		_ = server.MonitorConnections(
			&connection.MonitorScopeSelector{
				PathSegments: []*connection.PathSegment{{Name: segmentName}},
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
	connection.MonitorConnectionServer
}

func TestMonitorSendToRightClient(t *testing.T) {
	myServer := &myServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms)
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
	require.Equal(t, connection.ConnectionEventType_UPDATE, localMonitor.events[1].Type)
	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)
	// Just dummy update
	nsr.Connection.Context.IpContext.ExtraPrefixes = append(nsr.Connection.Context.IpContext.ExtraPrefixes, "10.2.3.1")
	_, _ = srv.Request(ctx, nsr)
	localMonitor.WaitEvents(timeoutCtx, 3)
	require.Equal(t, 3, len(localMonitor.events))
	require.Equal(t, connection.ConnectionEventType_UPDATE, localMonitor.events[2].Type)
	// check delete event is working fine.
	_, closeErr := srv.Close(ctx, nsr.Connection)
	require.Nil(t, closeErr)
	localMonitor.WaitEvents(timeoutCtx, 4)
	// Last event should be delete
	require.Equal(t, 4, len(localMonitor.events))
	require.Equal(t, connection.ConnectionEventType_DELETE, localMonitor.events[3].Type)
	// Check connection is not longer inside map
	require.Equal(t, len(mons.connections), 0)
}

func TestMonitorIsClosedProperly(t *testing.T) {
	myServer := &myServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms)

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
	require.Equal(t, connection.ConnectionEventType_UPDATE, localMonitor.events[1].Type)

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

type metricMonitorHolder struct {
	metricsMonitor MetricsMonitor
}

func (m *metricMonitorHolder) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	m.metricsMonitor = GetMetricsMonitor(ctx)
	return next.Server(ctx).Request(ctx, request)
}

func (m *metricMonitorHolder) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}

func TestSendMetrics(t *testing.T) {
	myServer := &myServer{}

	ctx := context.Background()
	ms := NewServer(&myServer.MonitorConnectionServer)
	mons, ok := ms.(*monitorServer)
	require.Equal(t, true, ok)

	holder := &metricMonitorHolder{}

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, holder)

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
	require.Equal(t, connection.ConnectionEventType_UPDATE, localMonitor.events[1].Type)

	// Check we have connection already
	require.Equal(t, len(mons.connections), 1)

	// Send metrics and check client are received it
	metrics := map[string]*connection.Metrics{
		nsr.Connection.Id: {
			Metrics: map[string]string{
				"rx": "10",
				"wx": "20",
			},
		},
		"some_other_conn": {
			Metrics: map[string]string{
				"rx": "10",
				"wx": "20",
			},
		},
	}
	holder.metricsMonitor.HandleMetrics(metrics)
	localMonitor.WaitEvents(timeoutCtx, 3)

	require.Equal(t, len(localMonitor.events[2].Metrics), 1)
}

func createConnection(id, server string) *connection.Connection {
	return &connection.Connection{
		Id: id,
		Path: &connection.Path{
			Index: 0,
			PathSegments: []*connection.PathSegment{
				{
					Name: server,
				},
			},
		},
		Context: &connectioncontext.ConnectionContext{
			IpContext: &connectioncontext.IPContext{
				SrcIpRequired: true,
				DstIpRequired: true,
			},
		},
	}
}
