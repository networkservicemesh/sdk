// Copyright (c) 2021-2025 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type connectionState int

const (
	zero connectionState = iota + 1
	established
	closed
)

var _ connectionState = zero

// ReselectFunc - function for changing request parameters on reselect
type ReselectFunc func(request *networkservice.NetworkServiceRequest)

// DefaultReselectFunc - default ReselectFunc
var DefaultReselectFunc ReselectFunc = func(request *networkservice.NetworkServiceRequest) {
	if request.GetConnection() != nil {
		request.GetConnection().Mechanism = nil
		request.GetConnection().NetworkServiceEndpointName = ""
		request.GetConnection().State = networkservice.State_RESELECT_REQUESTED
	}
}

// ReselectWithSameEndpointFunc - reselect function that doesn't clean NSE name
var ReselectWithSameEndpointFunc ReselectFunc = func(request *networkservice.NetworkServiceRequest) {
	if request.GetConnection() != nil {
		request.GetConnection().Mechanism = nil
		request.GetConnection().State = networkservice.State_RESELECT_REQUESTED
	}
}

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
	opts               []grpc.CallOption
	client             networkservice.NetworkServiceClient
	afterCloseFunc     func()
	reselectFunc       ReselectFunc
}

func newEventFactoryClient(ctx context.Context, afterClose func(), reselectFunc func(*networkservice.NetworkServiceRequest), opts ...grpc.CallOption) *eventFactoryClient {
	f := &eventFactoryClient{
		client:         next.Client(ctx),
		initialCtxFunc: postpone.Context(ctx),
		reselectFunc:   reselectFunc,
		opts:           opts,
	}
	f.updateContext(ctx)

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
		if f.state != established {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			request := f.request.Clone()
			if o.reselect {
				ctx, cancel := f.ctxFunc()
				_, _ = f.client.Close(ctx, request.GetConnection(), f.opts...)
				f.reselectFunc(request)
				cancel()
			}
			ctx, cancel := f.ctxFunc()
			defer cancel()
			conn, err := f.client.Request(ctx, request, f.opts...)
			if err == nil && f.request != nil {
				f.request.Connection = conn
				f.request.Connection.State = networkservice.State_UP
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
		if f.request == nil {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			ctx, cancel := f.ctxFunc()
			defer cancel()
			_, err := f.client.Close(ctx, f.request.GetConnection(), f.opts...)
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
	afterCloseFunc     func()
	server             networkservice.NetworkServiceServer
}

func newEventFactoryServer(ctx context.Context, afterClose func()) *eventFactoryServer {
	f := &eventFactoryServer{
		server:         next.Server(ctx),
		initialCtxFunc: postpone.Context(ctx),
	}
	f.updateContext(ctx)

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
		if f.state != established {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			ctx, cancel := f.ctxFunc()
			defer cancel()
			conn, err := f.server.Request(ctx, f.request)
			if err == nil && f.request != nil {
				f.request.Connection = conn
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
		if f.request == nil {
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
