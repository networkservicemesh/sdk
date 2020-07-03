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
	"sync"
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

type healClient struct {
	ctx            context.Context
	client         networkservice.MonitorConnectionClient
	onHeal         *networkservice.NetworkServiceClient
	heals          map[string]func()
	connections    map[string]*networkservice.Connection
	updateExecutor serialize.Executor
	eventLoopOnce  sync.Once
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that implements the healing algorithm
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
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
func NewClient(ctx context.Context, client networkservice.MonitorConnectionClient, onHeal *networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	rv := &healClient{
		ctx:            ctx,
		client:         client,
		onHeal:         onHeal,
		heals:          make(map[string]func()),
		connections:    make(map[string]*networkservice.Connection),
		updateExecutor: serialize.NewExecutor(),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(rv)
	}

	return rv
}

func (f *healClient) monitorLoop() {
	go f.eventLoopOnce.Do(func() {
		for {
			select {
			case <-f.ctx.Done():
				return
			default:
			}
			recv, err := f.client.MonitorConnections(f.ctx, &networkservice.MonitorScopeSelector{}, grpc.WaitForReady(true))
			if err != nil {
				continue
			}
			f.eventloop(recv)
		}
	})
}

// Should only be called inside monitorLoop
func (f *healClient) eventloop(recv networkservice.MonitorConnection_MonitorConnectionsClient) {
	for {
		select {
		case <-f.ctx.Done():
			return
		default:
		}
		event, err := recv.Recv()
		if err != nil {
			f.updateExecutor.AsyncExec(func() {
				for id, heal := range f.heals {
					heal()
					delete(f.connections, id)
				}
			})
			break
		}
		f.updateExecutor.AsyncExec(func() {
			switch event.GetType() {
			case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
				f.connections = event.GetConnections()
				if event.GetConnections() == nil {
					f.connections = map[string]*networkservice.Connection{}
				}

			case networkservice.ConnectionEventType_UPDATE:
				for _, conn := range event.GetConnections() {
					f.connections[conn.GetId()] = conn
				}

			case networkservice.ConnectionEventType_DELETE:
				for _, conn := range event.GetConnections() {
					delete(f.connections, conn.GetId())
				}
			}

			for id, heal := range f.heals {
				if _, ok := f.connections[id]; !ok {
					heal()
				}
			}
		})
	}
}

func (f *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	f.monitorLoop()
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	// Clone the request
	req := request.Clone()
	// Set its connection to the returned connection we received
	req.Connection = rv

	// TODO handle deadline ok
	deadline, ok := ctx.Deadline()
	if !ok {
		// If we don't have a deadline, we can literally deadlock on our attempts to heal (or make initial requests)
		return nil, errors.Errorf("all requests require a context with a deadline")
	}
	duration := time.Until(deadline)
	f.updateExecutor.AsyncExec(func() {
		f.heals[req.GetConnection().GetId()] = func() {
			healCtx, cancel := context.WithTimeout(f.ctx, duration)
			defer cancel()
			healCtx = extend.WithValuesFromContext(healCtx, ctx)
			select {
			case <-healCtx.Done():
				return
			default:
			}
			// TODO wrap another span around this
			_, err := (*f.onHeal).Request(healCtx, req, opts...)
			if err != nil {
				trace.Log(healCtx).Errorf("Attempt to heal connection %s resulted in error: %+v", req.GetConnection().GetId(), err)
			}
		}
	})
	return rv, nil
}

func (f *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	f.monitorLoop()
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	f.updateExecutor.AsyncExec(func() {
		delete(f.heals, conn.GetId())
		delete(f.connections, conn.GetId())
	})
	return rv, nil
}
