package monitor_test

import (
	"context"
	"net"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func BufConContextDialer(l *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, _ string) (net.Conn, error) {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())
		default:
		}
		return l.Dial()
	}
}

type eventConsumer struct {
	ch chan *networkservice.ConnectionEvent
}

func newEventConsumer() *eventConsumer {
	return &eventConsumer{
		ch: make(chan *networkservice.ConnectionEvent, 10),
	}
}

func (e *eventConsumer) Send(event *networkservice.ConnectionEvent) (err error) {
	e.ch <- event
	return nil
}

func (e *eventConsumer) Recv() (*networkservice.ConnectionEvent, error) {
	select {
	case event := <-e.ch:
		return event, nil
	default:
		return nil, errors.New("no event available")
	}
}

var _ monitor.EventConsumer = &eventConsumer{}

func TestMonitorClient_Request(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ec := newEventConsumer()
	ctx = monitor.WithEventConsumer(ctx, ec)
	defer cancel()
	l := bufconn.Listen(1024 * 1024)
	tokengen := sandbox.GenerateExpiringToken(time.Second)
	c := client.NewClient(
		ctx,
		client.WithClientURL(&url.URL{}),
		client.WithAuthorizeClient(null.NewClient()),
		client.WithDialOptions(
			append(
				sandbox.DialOptions(),
				grpc.WithContextDialer(BufConContextDialer(l)),
			)...,
		),
	)
	s := endpoint.NewServer(
		ctx,
		tokengen,
		endpoint.WithAuthorizeServer(null.NewServer()),
	)
	gsrv := grpc.NewServer()
	s.Register(gsrv)
	go func() { assert.NoError(t, gsrv.Serve(l)) }()
	conn, err := c.Request(ctx, &networkservice.NetworkServiceRequest{})
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	// TODO - look at using require.Never
	select {
	case event := <-ec.ch:
		require.Failf(t, "received event improperly", ": %+v", event)
	default:
	}
	_, err = s.Close(ctx, conn)
	assert.NoError(t, err)
	time.Sleep(10 * time.Second)
	runtime.Gosched()
	// TODO - look at using require.Eventually
	select {
	case event := <-ec.ch:
		require.Len(t, event.GetConnections(), 1)
		require.Equal(t, event.GetConnections()[conn.GetId()].GetState(), networkservice.State_DOWN)
	default:
		require.Fail(t, "failed to receive expected event")
	}
	gsrv.Stop()
}
