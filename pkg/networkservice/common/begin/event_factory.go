// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

package begin

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type errorEventFactoryInconsistency struct{}

func (e *errorEventFactoryInconsistency) Error() string {
	return "errorEventFactoryInconsistency error"
}

type connectionState int

const (
	zero connectionState = iota + 1
	established
	closed
)

var _ connectionState = zero

// EventFactory - allows firing off a Request or Close event from midchain
type EventFactory interface {
	Request(opts ...Option) <-chan error
	Close(opts ...Option) <-chan error
}

type eventFactoryClient struct {
	state              connectionState
	executor           serialize.Executor
	initialCtxFunc     func() (context.Context, context.CancelFunc)
	ctxFunc            func() (context.Context, context.CancelFunc)
	request            *networkservice.NetworkServiceRequest
	returnedConnection *networkservice.Connection
	grpcOpts           []grpc.CallOption
	client             networkservice.NetworkServiceClient
	beforeRequestFunc  func() error
	afterCloseFunc     func()
}

func newEventFactoryClient(ctx context.Context, actualEventFactoryFunc func() *eventFactoryClient, afterClose func()) *eventFactoryClient {
	f := &eventFactoryClient{
		client:         next.Client(ctx),
		initialCtxFunc: postpone.Context(ctx),
	}
	f.updateContext(ctx)

	f.beforeRequestFunc = func() error {
		if actualEventFactoryFunc != nil && actualEventFactoryFunc() != f {
			return &errorEventFactoryInconsistency{}
		}
		return nil
	}

	f.afterCloseFunc = func() {
		f.state = closed
		if afterClose != nil {
			afterClose()
		}
	}
	return f
}

func (f *eventFactoryClient) updateContext(valueCtx context.Context) {
	f.ctxFunc = func() (context.Context, context.CancelFunc) {
		eventCtx, cancel := f.initialCtxFunc()
		eventCtx = extend.WithValuesFromContext(eventCtx, valueCtx)
		return withEventFactory(eventCtx, f), cancel
	}
}

func (f *eventFactoryClient) Request(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}
	ch := make(chan error, 1)

	f.executor.AsyncExec(func() {
		defer close(ch)
		if err := f.beforeRequestFunc(); err != nil {
			ch <- err
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			request := f.request.Clone()
			grpcOpts := f.grpcOpts
			if o.userRequest != nil {
				request = o.userRequest
				request.Connection = mergeConnection(f.returnedConnection, o.userRequest.Connection, f.request.GetConnection())
			}
			if o.grpcOpts != nil {
				grpcOpts = o.grpcOpts
			}

			if o.reselect {
				ctx, cancel := f.ctxFunc()
				_, _ = f.client.Close(ctx, request.GetConnection(), grpcOpts...)
				if request.GetConnection() != nil {
					request.GetConnection().Mechanism = nil
					request.GetConnection().NetworkServiceEndpointName = ""
					request.GetConnection().State = networkservice.State_RESELECT_REQUESTED
				}
				cancel()
			}

			var ctx context.Context
			if o.ctx != nil {
				ctx = withEventFactory(o.ctx, f)
			} else {
				var cancel context.CancelFunc
				ctx, cancel = f.ctxFunc()
				defer cancel()
			}

			conn, err := f.client.Request(ctx, request, grpcOpts...)
			if err == nil {
				f.request = request.Clone()
				f.request.Connection = conn.Clone()
				f.grpcOpts = grpcOpts
				f.state = established
				f.request.Connection.State = networkservice.State_UP
				if o.connectionToReturn != nil {
					f.returnedConnection = conn.Clone()
					*o.connectionToReturn = conn.Clone()
				}
				f.updateContext(ctx)
			} else if f.state != established {
				f.afterCloseFunc()
			}
			ch <- err
		}
	})
	return ch
}

func (f *eventFactoryClient) Close(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}

	ch := make(chan error, 1)
	f.executor.AsyncExec(func() {
		defer close(ch)
		if f.request == nil || f.state != established {
			return
		}

		select {
		case <-o.cancelCtx.Done():
		default:
			grpcOpts := f.grpcOpts
			if o.grpcOpts != nil {
				grpcOpts = o.grpcOpts
			}

			ctx, cancel := f.ctxFunc()
			defer cancel()

			_, err := f.client.Close(ctx, f.request.GetConnection(), grpcOpts...)
			f.afterCloseFunc()
			ch <- err
		}
	})
	return ch
}

var _ EventFactory = &eventFactoryClient{}

type eventFactoryServer struct {
	state              connectionState
	executor           serialize.Executor
	initialCtxFunc     func() (context.Context, context.CancelFunc)
	ctxFunc            func() (context.Context, context.CancelFunc)
	request            *networkservice.NetworkServiceRequest
	returnedConnection *networkservice.Connection
	server             networkservice.NetworkServiceServer
	beforeRequestFunc  func() error
	afterCloseFunc     func()
}

func newEventFactoryServer(ctx context.Context, actualEventFactoryFunc func() *eventFactoryServer, afterClose func()) *eventFactoryServer {
	f := &eventFactoryServer{
		server:         next.Server(ctx),
		initialCtxFunc: postpone.Context(ctx),
	}
	f.updateContext(ctx)

	f.beforeRequestFunc = func() error {
		if actualEventFactoryFunc != nil && actualEventFactoryFunc() != f {
			return &errorEventFactoryInconsistency{}
		}
		return nil
	}

	f.afterCloseFunc = func() {
		f.state = closed
		afterClose()
	}
	return f
}

func (f *eventFactoryServer) updateContext(valueCtx context.Context) {
	f.ctxFunc = func() (context.Context, context.CancelFunc) {
		eventCtx, cancel := f.initialCtxFunc()
		eventCtx = extend.WithValuesFromContext(eventCtx, valueCtx)
		eventCtx = peer.NewContext(eventCtx, &peer.Peer{})
		return withEventFactory(eventCtx, f), cancel
	}
}

func (f *eventFactoryServer) Request(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}
	ch := make(chan error, 1)
	f.executor.AsyncExec(func() {
		defer close(ch)
		if err := f.beforeRequestFunc(); err != nil {
			ch <- err
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			request := f.request.Clone()
			if o.userRequest != nil {
				request = o.userRequest
			}
			if f.state == established && request.GetConnection().GetState() == networkservice.State_RESELECT_REQUESTED {
				ctx, cancel := f.ctxFunc()
				_, _ = f.server.Close(ctx, f.request.GetConnection())
				f.state = closed
				cancel()
			}

			var ctx context.Context
			if o.ctx != nil {
				ctx = withEventFactory(o.ctx, f)
			} else {
				var cancel context.CancelFunc
				ctx, cancel = f.ctxFunc()
				defer cancel()
			}

			conn, err := f.server.Request(ctx, request)
			if err == nil {
				f.request = request.Clone()
				f.request.Connection = conn.Clone()
				f.state = established
				f.request.Connection.State = networkservice.State_UP
				if o.connectionToReturn != nil {
					f.returnedConnection = conn.Clone()
					*o.connectionToReturn = conn.Clone()
				}
				f.updateContext(ctx)
			} else if f.state != established {
				f.afterCloseFunc()
			}
			ch <- err
		}
	})
	return ch
}

func (f *eventFactoryServer) Close(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}
	ch := make(chan error, 1)
	f.executor.AsyncExec(func() {
		defer close(ch)
		if f.request == nil || f.state != established {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			ctx, cancel := f.ctxFunc()
			defer cancel()
			_, err := f.server.Close(ctx, f.request.GetConnection())
			f.afterCloseFunc()
			ch <- err
		}
	})
	return ch
}

var _ EventFactory = &eventFactoryServer{}
