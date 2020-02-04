package metrics

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/stretchr/testify/require"
)

type metricMonitorHolder struct {
	index int
	networkservice.MonitorConnection_MonitorConnectionsServer
	events       []*networkservice.ConnectionEvent
	monitor      MetricsMonitor
	server       networkservice.MonitorConnection_MonitorConnectionsServer
	eventChannel chan *networkservice.ConnectionEvent
}

func (t *metricMonitorHolder) Send(event *networkservice.ConnectionEvent) error {
	t.events = append(t.events, event)
	t.eventChannel <- event
	return nil
}

func newMetricMonitorHolder() *metricMonitorHolder {
	return &metricMonitorHolder{
		index:        0,
		eventChannel: make(chan *networkservice.ConnectionEvent, 100),
	}
}

func (t *metricMonitorHolder) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	t.index += 1
	t.monitor = Server(ctx)
	t.server = monitor.Server(ctx)

	request.GetConnection().Labels = make(map[string]string)
	request.GetConnection().Labels["lastUpdate"] = fmt.Sprintf("%v %v", t.index, time.Now())
	return request.GetConnection(), nil
}

func (t *metricMonitorHolder) Close(context.Context, *networkservice.Connection) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

func (t *metricMonitorHolder) WaitEvents(ctx context.Context, count int) {
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

func TestSendMetrics(t *testing.T) {
	ctx := context.Background()

	timeoutCtx, _ := context.WithTimeout(context.Background(), time.Second*5)

	ms := NewServer()
	mons, ok := ms.(*metricsServer)
	require.Equal(t, true, ok)

	holder := newMetricMonitorHolder()

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, holder)

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", 1, []string{"local-nsm", "remote-nsm"}, int64(1 * time.Millisecond)),
	}

	ctx = monitor.WithServer(ctx, holder)
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.
	require.Equal(t, len(mons.metrics), 1)

	<-time.After(1 * time.Millisecond)
	holder.monitor.HandleMetrics(map[string]string{
		"rx": "100",
		"wx": "100",
	})

	holder.WaitEvents(timeoutCtx, 1)
	require.Equal(t, len(holder.events), 1)

	require.NotNil(t, holder.events[0].Metrics)

	m := holder.events[0].Metrics["id0"]
	require.Equal(t, m.MetricsSegment[0].Name, "local-nsm")
	require.Nil(t, m.MetricsSegment[0].Metrics)

	require.Equal(t, m.MetricsSegment[1].Name, "remote-nsm")
	require.NotNil(t, m.MetricsSegment[1].Metrics)
	require.Equal(t, len(m.MetricsSegment[1].Metrics), 2)
}

func TestIntervalSendMetrics(t *testing.T) {
	ctx := context.Background()

	//timeoutCtx, _ := context.WithTimeout(context.Background(), time.Second*5)

	ms := NewServer()
	mons, ok := ms.(*metricsServer)
	require.Equal(t, true, ok)

	holder := newMetricMonitorHolder()

	srv := next.NewWrappedNetworkServiceServer(trace.NewNetworkServiceServer, ms, holder)

	// Add first connection and check right listener has event
	// After it will think connection is established.
	nsr := &networkservice.NetworkServiceRequest{
		Connection: createConnection("id0", 1, []string{"local-nsm", "remote-nsm"}, int64(5 * time.Hour)),
	}

	ctx = monitor.WithServer(ctx, holder)
	_, _ = srv.Request(ctx, nsr)
	// Now we could check monitoring routine's are working fine.
	require.Equal(t, len(mons.metrics), 1)

	<-time.After(1 * time.Millisecond)
	holder.monitor.HandleMetrics(map[string]string{
		"rx": "100",
		"wx": "100",
	})

	// There is no event yet
	require.Equal(t, len(holder.events), 0)

	// but if we reqest again, we will have event send.
	_, _ = srv.Request(ctx, nsr)
	<- time.After(60 * time.Second)
	require.NotNil(t, holder.events[0].Metrics)

	m := holder.events[0].Metrics["id0"]
	require.Equal(t, m.MetricsSegment[0].Name, "local-nsm")
	require.Nil(t, m.MetricsSegment[0].Metrics)

	require.Equal(t, m.MetricsSegment[1].Name, "remote-nsm")
	require.NotNil(t, m.MetricsSegment[1].Metrics)
	require.Equal(t, len(m.MetricsSegment[1].Metrics), 2)
}


func createConnection(id string, index uint32, segments []string, interval int64) *networkservice.Connection {
	result := &networkservice.Connection{
		Id: id,
		Path: &networkservice.Path{
			Index: index,
			PathSegments: []*networkservice.PathSegment{
			},
		},
		Context: &networkservice.ConnectionContext{
			IpContext: &networkservice.IPContext{
				SrcIpRequired: true,
				DstIpRequired: true,
			},
			MetricsContext: &networkservice.MetricsContext{
				Enabled:  true,
				Interval: interval, // Make it really/ really small
			},
		},
	}
	for _, s := range segments {
		result.Path.PathSegments = append(result.Path.PathSegments, &networkservice.PathSegment{
			Id:   s,
			Name: s,
		})
	}
	return result
}
