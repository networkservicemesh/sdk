// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package heal provides a chain element that carries out proper nsm healing from client to endpoint
package heal

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type healClient struct {
	requestors        map[string]func()
	closers           map[string]func()
	reported          map[string]*networkservice.Connection
	onHeal            *networkservice.NetworkServiceClient
	client            networkservice.MonitorConnectionClient
	eventReceiver     networkservice.MonitorConnection_MonitorConnectionsClient
	updateExecutor    serialize.Executor
	recvEventExecutor serialize.Executor
	cancelFunc        context.CancelFunc
	connectionIDs     map[string]bool
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that implements the healing algorithm
//             - client - networkservice.MonitorConnectionClient that can be used to call MonitorConnection against the endpoint
//             - onHeal - *networkservice.NetworkServiceClient.  Since networkservice.NetworkServiceClient is an interface
//                        (and thus a pointer) *networkservice.NetworkServiceClient is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.NetworkServiceClient
//                        This is done because when we use heal.NewClient as part of a chain, we may not *have*
//                        a pointer to this
//                        client used 'onHeal'.  If we detect we need to heal, onHeal.Request is used to heal.
//                        If onHeal is nil, then we simply set onHeal to this client chain element
//                        If we are part of a larger chain or a server, we should pass the resulting chain into
//                        this constructor before we actually have a pointer to it.
//                        If onHeal nil, onHeal will be pointed to the returned networkservice.NetworkServiceClient
func NewClient(client networkservice.MonitorConnectionClient, onHeal *networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	rv := &healClient{
		onHeal:            onHeal,
		requestors:        make(map[string]func()),
		closers:           make(map[string]func()),
		reported:          make(map[string]*networkservice.Connection),
		connectionIDs:     make(map[string]bool),
		client:            client,
		updateExecutor:    serialize.NewExecutor(),
		eventReceiver:     nil, // This is intentionally nil
		recvEventExecutor: serialize.NewExecutor(),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(rv)
	}

	return rv
}

func (f *healClient) init() {
	if f.eventReceiver != nil {
		select {
		case <-f.eventReceiver.Context().Done():
			return
		default:
			f.eventReceiver = nil
		}
	}

	logrus.Info("Creating new eventReceiver")

	ctx, cancelFunc := context.WithCancel(context.Background())
	f.cancelFunc = cancelFunc

	recv, err := f.client.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{}, grpc.WaitForReady(true))
	if err != nil {
		f.updateExecutor.AsyncExec(f.init)
		return
	}

	f.eventReceiver = recv
	f.recvEventExecutor.AsyncExec(f.recvEvent)
}

func (f *healClient) lazyInit(id string) {
	f.updateExecutor.SyncExec(func() {
		if _, ok := f.connectionIDs[id]; !ok {
			f.connectionIDs[id] = true
			f.updateExecutor.AsyncExec(f.init)
		}
	})
}

func (f *healClient) tearDown() {
	if f.cancelFunc != nil {
		f.cancelFunc()
	}
}

func (f *healClient) lazyTearDown(id string) {
	f.updateExecutor.SyncExec(func() {
		delete(f.connectionIDs, id)
		if _, ok := f.connectionIDs[id]; !ok {
			f.updateExecutor.AsyncExec(f.tearDown)
		}
	})
}

func (f *healClient) recvEvent() {
	select {
	case <-f.eventReceiver.Context().Done():
		return
	default:
		event, err := f.eventReceiver.Recv()
		f.updateExecutor.AsyncExec(func() {
			if err != nil {
				for id := range f.reported {
					f.requestors[id]()
					delete(f.reported, id)
				}
				f.init()
				return
			}

			switch event.GetType() {
			case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
				f.reported = event.GetConnections()
				if event.GetConnections() == nil {
					f.reported = map[string]*networkservice.Connection{}
				}

			case networkservice.ConnectionEventType_UPDATE:
				for _, conn := range event.GetConnections() {
					f.reported[conn.GetId()] = conn
				}

			case networkservice.ConnectionEventType_DELETE:
				for id, conn := range event.GetConnections() {
					f.requestors[id]()
					delete(f.reported, conn.GetId())
				}

			}
		})
	}

	f.recvEventExecutor.AsyncExec(f.recvEvent)
}

func (f *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	f.lazyInit(request.GetConnection().GetId())

	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	// Clone the request
	req := request.Clone()
	// Set its connection to the returned connection we received
	req.Connection = rv

	// TODO handle deadline err
	deadline, _ := ctx.Deadline()
	duration := time.Until(deadline)
	f.updateExecutor.AsyncExec(func() {
		f.requestors[req.GetConnection().GetId()] = func() {
			timeCtx, cancelFunc := context.WithTimeout(context.Background(), duration)
			ctx = extend.WithValuesFromContext(timeCtx, ctx)
			// TODO wrap another span around this
			_, err := (*f.onHeal).Request(ctx, req, opts...)
			trace.Log(ctx).Errorf("Attempt to heal connection %s resulted in error: %+v", req.GetConnection().GetId(), err)
			cancelFunc()
		}
		f.closers[req.GetConnection().GetId()] = func() {
			timeCtx, cancelFunc := context.WithTimeout(context.Background(), duration)
			ctx = extend.WithValuesFromContext(timeCtx, ctx)
			_, err := (*f.onHeal).Close(extend.WithValuesFromContext(timeCtx, ctx), req.GetConnection(), opts...)
			trace.Log(ctx).Errorf("Attempt to close connection %s during heal resulted in error: %+v", req.GetConnection().GetId(), err)
			cancelFunc()
		}
	})
	return rv, nil
}

func (f *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	f.lazyTearDown(conn.GetId())

	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	f.updateExecutor.AsyncExec(func() {
		delete(f.requestors, conn.GetId())
		delete(f.closers, conn.GetId())
		delete(f.reported, conn.GetId())
	})
	return rv, nil
}
