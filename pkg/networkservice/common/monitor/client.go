package monitor

import (
	"context"
	"runtime"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type monitorClient struct {
	chainCtx context.Context
}

func NewClient() networkservice.NetworkServiceClient {
	return &monitorClient{}
}

func (m *monitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	go monitor(ctx, conn.Clone(), metadata.IsClient(m))
	return conn, err
}

func (m *monitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// Cancel any old monitor we have around
	if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		oldCancel()
	}
	return next.Client(ctx).Close(ctx, conn)
}

func monitor(ctx context.Context, conn *networkservice.Connection, isClient bool) {
	// Is another chain element asking for events
	eventConsumers := FromContext(ctx)
	if eventConsumers == nil {
		return
	}
	// Do we have a grpc.ClientConnInterface?
	cc, ccLoaded := clientconn.Load(ctx, isClient)
	if !ccLoaded {
		return
	}

	// Cancel any old monitor we have around
	if oldCancel, loaded := loadAndDelete(ctx, isClient); loaded {
		oldCancel()
	}

	// Create and store a new monitorCtx
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	store(ctx, isClient, monitorCancel)

	eventLoop(monitorCtx, networkservice.NewMonitorConnectionClient(cc), eventConsumers, conn)
}

func eventLoop(monitorCtx context.Context, monitorConnectionClient networkservice.MonitorConnectionClient, eventConsumers []EventConsumer, conn *networkservice.Connection) {
	selector := &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Id:   conn.GetCurrentPathSegment().GetId(),
				Name: conn.GetCurrentPathSegment().GetName(),
			},
		},
	}
	for {
		select {
		case <-monitorCtx.Done():
			return
		default:
		}
		client, err := monitorConnectionClient.MonitorConnections(monitorCtx, selector)

		for err == nil {
			var eventIn *networkservice.ConnectionEvent
			eventIn, err = client.Recv()
			if err != nil || eventIn == nil {
				continue
			}

			select {
			case <-monitorCtx.Done():
				return
			default:
			}

			eventOut := &networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: make(map[string]*networkservice.Connection),
			}

			for _, connIn := range eventIn.GetConnections() {

				// Make sure our subsequent manipulations will be safe
				if connIn == nil || connIn.GetPath() == nil || connIn.GetPath().GetIndex() < 1 {
					continue
				}
				// Add the Connection to the outgoing event, and step its index back 1 to point to the index of this client
				connOut := connIn.Clone()
				connOut.Id = connIn.GetPrevPathSegment().GetId()
				connOut.GetPath().Index--

				// If this isn't for our conn... we shouldn't have gotten it... but maybe the server is buggy... skip this one
				if connOut.GetId() != conn.GetId() {
					continue
				}

				// If it's deleted, mark the event state down
				if eventIn.GetType() == networkservice.ConnectionEventType_DELETE {
					connOut.State = networkservice.State_DOWN
				}

				// If this is exactly our conn... it's not news, no need to pass it on
				if connOut.Equals(conn) {
					continue
				}

				// If we've gotten here, it's a good connection to pass on :)
				eventOut.GetConnections()[connOut.GetId()] = connOut
			}

			// If we have surviving connections about which to report an event, send it out to the EventConsumers
			if len(eventOut.GetConnections()) > 0 {
				for _, eventConsumer := range eventConsumers {
					go func(eventConsumer EventConsumer) { _ = eventConsumer.Send(eventOut) }(eventConsumer)
				}
			}
		}
		runtime.Gosched()
	}
}
