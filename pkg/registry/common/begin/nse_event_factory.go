// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type eventNSEFactoryClient struct {
	state          connectionState
	executor       serialize.Executor
	ctxFunc        func() (context.Context, context.CancelFunc)
	registration   *registry.NetworkServiceEndpoint
	response       *registry.NetworkServiceEndpoint
	opts           []grpc.CallOption
	client         registry.NetworkServiceEndpointRegistryClient
	afterCloseFunc func()
}

func newEventNSEFactoryClient(ctx context.Context, afterClose func(), opts ...grpc.CallOption) *eventNSEFactoryClient {
	f := &eventNSEFactoryClient{
		client: next.NetworkServiceEndpointRegistryClient(ctx),
		opts:   opts,
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

func (f *eventNSEFactoryClient) updateContext(ctx context.Context) {
	ctxFunc := postpone.ContextWithValues(ctx)
	f.ctxFunc = func() (context.Context, context.CancelFunc) {
		eventCtx, cancel := ctxFunc()
		return withEventFactory(eventCtx, f), cancel
	}
}

func (f *eventNSEFactoryClient) Register(opts ...Option) <-chan error {
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
			registration := f.registration.Clone()
			ctx, cancel := f.ctxFunc()
			defer cancel()
			resp, err := f.client.Register(ctx, registration, f.opts...)
			if err == nil && f.registration != nil {
				f.registration = mergeNSE(f.registration, resp)
			}
			ch <- err
		}
	})
	return ch
}

func (f *eventNSEFactoryClient) Unregister(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}
	ch := make(chan error, 1)
	f.executor.AsyncExec(func() {
		defer close(ch)
		if f.registration == nil {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			ctx, cancel := f.ctxFunc()
			defer cancel()
			_, err := f.client.Unregister(ctx, f.response, f.opts...)
			f.afterCloseFunc()
			ch <- err
		}
	})
	return ch
}

var _ EventFactory = &eventNSEFactoryClient{}

type eventNSEFactoryServer struct {
	state          connectionState
	executor       serialize.Executor
	ctxFunc        func() (context.Context, context.CancelFunc)
	registration   *registry.NetworkServiceEndpoint
	response       *registry.NetworkServiceEndpoint
	afterCloseFunc func()
	server         registry.NetworkServiceEndpointRegistryServer
}

func newNSEEventFactoryServer(ctx context.Context, afterClose func()) *eventNSEFactoryServer {
	f := &eventNSEFactoryServer{
		server: next.NetworkServiceEndpointRegistryServer(ctx),
	}
	f.updateContext(ctx)

	f.afterCloseFunc = func() {
		f.state = closed
		afterClose()
	}
	return f
}

func (f *eventNSEFactoryServer) updateContext(ctx context.Context) {
	ctxFunc := postpone.ContextWithValues(ctx)
	f.ctxFunc = func() (context.Context, context.CancelFunc) {
		eventCtx, cancel := ctxFunc()
		return withEventFactory(eventCtx, f), cancel
	}
}

func (f *eventNSEFactoryServer) Register(opts ...Option) <-chan error {
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
			resp, err := f.server.Register(ctx, f.registration)
			if err == nil && f.registration != nil {
				f.registration = resp
			}
			ch <- err
		}
	})
	return ch
}

func (f *eventNSEFactoryServer) Unregister(opts ...Option) <-chan error {
	o := &option{
		cancelCtx: context.Background(),
	}
	for _, opt := range opts {
		opt(o)
	}
	ch := make(chan error, 1)
	f.executor.AsyncExec(func() {
		defer close(ch)
		if f.registration == nil {
			return
		}
		select {
		case <-o.cancelCtx.Done():
		default:
			ctx, cancel := f.ctxFunc()
			defer cancel()
			_, err := f.server.Unregister(ctx, f.registration)
			f.afterCloseFunc()
			ch <- err
		}
	})
	return ch
}

var _ EventFactory = &eventNSEFactoryServer{}
