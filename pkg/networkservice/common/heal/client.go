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
	"github.com/sirupsen/logrus"
	"time"

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

type HealClient struct {
	requestors        map[string]func()
	closers           map[string]func()
	reported          map[string]*networkservice.Connection
	onHeal            *networkservice.NetworkServiceClient
	client            networkservice.MonitorConnectionClient
	eventReceiver     networkservice.MonitorConnection_MonitorConnectionsClient
	updateExecutor    serialize.Executor
	recvEventExecutor serialize.Executor
	cancelFunc        context.CancelFunc
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
	rv := &HealClient{
		onHeal:            onHeal,
		requestors:        make(map[string]func()),
		closers:           make(map[string]func()),
		reported:          make(map[string]*networkservice.Connection),
		client:            client,
		updateExecutor:    serialize.NewExecutor(),
		eventReceiver:     nil, // This is intentionally nil
		recvEventExecutor: serialize.NewExecutor(),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(rv)
	}

	rv.recvEventExecutor.AsyncExec(rv.recvEvent)

	return rv
}

func (f *HealClient) recvEvent() {
	if f.eventReceiver == nil {
		logrus.Info("creating new eventReceiver")

		ctx, cancelFunc := context.WithCancel(context.Background())
		f.updateExecutor.AsyncExec(func() {
			f.cancelFunc = cancelFunc
		})

		recv, err := f.client.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{}, grpc.WaitForReady(true))
		if err != nil {
			f.recvEventExecutor.AsyncExec(f.recvEvent)
			return
		}

		f.eventReceiver = recv
	}

	select {
	case <-f.eventReceiver.Context().Done():
		f.resetEventReceiver()
		return
	default:
		event, err := f.eventReceiver.Recv()
		f.updateExecutor.AsyncExec(func() {
			switch {
			case err != nil:
				f.resetEventReceiver()
				for k := range f.reported {
					delete(f.reported, k)
				}
			case event.GetType() == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
				f.reported = event.GetConnections()
			case event.GetType() == networkservice.ConnectionEventType_UPDATE:
				for _, conn := range event.GetConnections() {
					f.reported[conn.GetId()] = conn
				}
			case event.GetType() == networkservice.ConnectionEventType_DELETE:
				for _, conn := range event.GetConnections() {
					delete(f.reported, conn.GetId())
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

func (f *HealClient) resetEventReceiver() {
	f.recvEventExecutor.AsyncExec(func() {
		f.eventReceiver = nil
	})
}

func (f *HealClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
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

func (f *HealClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	f.updateExecutor.AsyncExec(func() {
		delete(f.requestors, conn.GetId())
		delete(f.closers, conn.GetId())
	})
	return rv, nil
}

func (f *HealClient) Stop() {
	f.updateExecutor.SyncExec(func() {
		if f.cancelFunc != nil {
			f.cancelFunc()
		}
	})
}
