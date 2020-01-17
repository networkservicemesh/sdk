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
	"runtime"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/next"
)

type healClient struct {
	requestors        map[string]func()
	closers           map[string]func()
	reported          map[string]*connection.Connection
	onHeal            *networkservice.NetworkServiceClient
	client            connection.MonitorConnectionClient
	eventReceiver     connection.MonitorConnection_MonitorConnectionsClient
	updateExecutor    serialize.Executor
	recvEventExecutor serialize.Executor
	cancelFunc        context.CancelFunc
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that implements the healing algorithm
//             - client - connection.MonitorConnectionClient that can be used to call MonitorConnection against the endpoint
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
func NewClient(client connection.MonitorConnectionClient, onHeal *networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	rv := &healClient{
		onHeal:            onHeal,
		requestors:        make(map[string]func()),
		closers:           make(map[string]func()),
		reported:          make(map[string]*connection.Connection),
		client:            client,
		updateExecutor:    serialize.NewExecutor(),
		eventReceiver:     nil, // This is intentionally nil
		recvEventExecutor: serialize.NewExecutor(),
	}
	if rv.onHeal == nil {
		var actualOnHeal networkservice.NetworkServiceClient = rv
		rv.onHeal = &actualOnHeal
	}
	rv.updateExecutor.AsyncExec(func() {
		runtime.SetFinalizer(rv, func(f *healClient) {
			f.updateExecutor.AsyncExec(func() {
				if f.cancelFunc != nil {
					f.cancelFunc()
				}
			})
		})
		ctx, cancelFunc := context.WithCancel(context.Background())
		rv.cancelFunc = cancelFunc
		// TODO decide what to do about err here
		recv, _ := rv.client.MonitorConnections(ctx, nil)
		rv.eventReceiver = recv
		rv.recvEventExecutor.AsyncExec(rv.recvEvent)
	})

	return rv
}

func (f *healClient) recvEvent() {
	select {
	case <-f.eventReceiver.Context().Done():
		f.eventReceiver = nil
	default:
		event, err := f.eventReceiver.Recv()
		if err != nil {
			event = nil
		}
		f.updateExecutor.AsyncExec(func() {
			switch event.GetType() {
			case connection.ConnectionEventType_INITIAL_STATE_TRANSFER:
				f.reported = event.GetConnections()
			case connection.ConnectionEventType_UPDATE:
				for _, conn := range event.GetConnections() {
					f.reported[conn.GetId()] = conn
				}
			case connection.ConnectionEventType_DELETE:
				for _, conn := range event.GetConnections() {
					delete(f.reported, conn.GetId())
					if f.requestors[conn.GetId()] != nil {
						f.requestors[conn.GetId()]()
					}
				}
			}
			for id, request := range f.requestors {
				if _, ok := f.reported[id]; !ok {
					request()
				}
			}
		})
	}
	f.recvEventExecutor.AsyncExec(f.recvEvent)
}

func (f *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*connection.Connection, error) {
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

func (f *healClient) Close(ctx context.Context, conn *connection.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	f.updateExecutor.AsyncExec(func() {
		delete(f.requestors, conn.GetId())
	})
	return rv, nil
}
