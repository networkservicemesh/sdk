// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2021 Doc.ai and/or its affiliates.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ctxWrapper struct {
	mut     sync.Mutex
	request *networkservice.NetworkServiceRequest
	ctx     context.Context
	cancel  func()
}

type healServer struct {
	ctx            context.Context
	onHeal         *networkservice.NetworkServiceClient
	onRestore      OnRestore
	restoreTimeout time.Duration
	healContextMap ctxWrapperMap
}

// NewServer - creates a new networkservice.NetworkServiceServer chain element that implements the healing algorithm
func NewServer(ctx context.Context, opts ...Option) networkservice.NetworkServiceServer {
	healOpts := &healOptions{
		onRestore:      OnRestoreHeal,
		restoreTimeout: time.Minute,
	}
	for _, opt := range opts {
		opt(healOpts)
	}

	rv := &healServer{
		ctx:            ctx,
		onHeal:         healOpts.onHeal,
		onRestore:      healOpts.onRestore,
		restoreTimeout: healOpts.restoreTimeout,
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(adapters.NewServerToClient(rv))
	}

	return rv
}

func (f *healServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = f.withHandlers(ctx)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	cw, loaded := f.healContextMap.LoadOrStore(request.GetConnection().GetId(), &ctxWrapper{
		request: request.Clone(),
		ctx:     f.createHealContext(ctx, nil),
	})
	if loaded {
		cw.mut.Lock()
		defer cw.mut.Unlock()

		if cw.cancel != nil {
			cw.cancel()
			cw.cancel = nil
		}
		cw.request = request.Clone()
		cw.ctx = f.createHealContext(ctx, cw.ctx)
	}

	return conn, nil
}

func (f *healServer) withHandlers(ctx context.Context) context.Context {
	ctx = withRequestHealConnectionFunc(ctx, f.handleHealConnectionRequest)

	var restoreConnectionHandler requestHealFuncType
	switch f.onRestore {
	case OnRestoreRestore:
		restoreConnectionHandler = f.handleRestoreConnectionRequest
	case OnRestoreHeal:
		restoreConnectionHandler = f.handleHealConnectionRequest
	case OnRestoreIgnore:
		restoreConnectionHandler = func(*networkservice.Connection) {}
	}
	ctx = withRequestRestoreConnectionFunc(ctx, restoreConnectionHandler)

	return ctx
}

func (f *healServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := next.Server(ctx).Close(ctx, conn)

	f.stopHeal(conn)

	return rv, err
}

func (f *healServer) getHealContext(conn *networkservice.Connection) (*networkservice.NetworkServiceRequest, context.Context) {
	cw, ok := f.healContextMap.Load(conn.GetId())
	if !ok {
		return nil, nil
	}

	cw.mut.Lock()
	defer cw.mut.Unlock()

	if cw.cancel != nil {
		cw.cancel()
	}
	ctx, cancel := context.WithCancel(cw.ctx)
	cw.cancel = cancel
	request := cw.request.Clone()

	return request, ctx
}

// handleHealConnectionRequest - heals requested connection. Returns immediately, heal is asynchronous.
func (f *healServer) handleHealConnectionRequest(conn *networkservice.Connection) {
	request, healCtx := f.getHealContext(conn)
	if request == nil {
		return
	}

	request.SetRequestConnection(conn.Clone())

	go f.processHeal(healCtx, request)
}

// handleRestoreConnectionRequest - recreates connection. Returns immediately, heal is asynchronous.
func (f *healServer) handleRestoreConnectionRequest(conn *networkservice.Connection) {
	request, healCtx := f.getHealContext(conn)
	if request == nil {
		return
	}

	request.SetRequestConnection(conn.Clone())

	go f.restoreConnection(healCtx, request)
}

func (f *healServer) stopHeal(conn *networkservice.Connection) {
	cw, loaded := f.healContextMap.LoadAndDelete(conn.GetId())
	if !loaded {
		return
	}
	cw.mut.Lock()
	defer cw.mut.Unlock()
	if cw.cancel != nil {
		cw.cancel()
	}
}

func (f *healServer) restoreConnection(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
	clockTime := clock.FromContext(ctx)

	if ctx.Err() != nil {
		return
	}

	// Make sure we have a valid expireTime to work with
	expires := request.GetConnection().GetNextPathSegment().GetExpires()
	if !expires.IsValid() {
		return
	}
	expireTime := expires.AsTime()

	deadline := clockTime.Now().Add(f.restoreTimeout)
	if deadline.After(expireTime) {
		deadline = expireTime
	}
	requestCtx, requestCancel := clockTime.WithDeadline(ctx, deadline)
	defer requestCancel()

	for requestCtx.Err() == nil {
		if _, err := (*f.onHeal).Request(requestCtx, request.Clone(), opts...); err == nil {
			return
		}
	}

	f.processHeal(ctx, request.Clone(), opts...)
}

func (f *healServer) processHeal(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
	logEntry := log.FromContext(ctx).WithField("healServer", "processHeal")
	conn := request.GetConnection()

	if ctx.Err() != nil {
		return
	}

	candidates := discover.Candidates(ctx)
	if candidates != nil || conn.GetPath().GetIndex() == 0 {
		logEntry.Infof("Starting heal process for %s", conn.GetId())

		healCtx, healCancel := context.WithCancel(ctx)
		defer healCancel()

		reRequest := request.Clone()
		reRequest.GetConnection().NetworkServiceEndpointName = ""
		path := reRequest.GetConnection().Path
		reRequest.GetConnection().Path.PathSegments = path.PathSegments[0 : path.Index+1]

		for ctx.Err() == nil {
			_, err := (*f.onHeal).Request(healCtx, reRequest, opts...)
			if err != nil {
				logEntry.Errorf("Failed to heal connection %s: %v", conn.GetId(), err)
			} else {
				logEntry.Infof("Finished heal process for %s", conn.GetId())
				break
			}
		}
	} else {
		// Huge timeout is not required to close connection on a current path segment
		closeCtx, closeCancel := clock.FromContext(ctx).WithTimeout(ctx, time.Second)
		defer closeCancel()

		_, err := (*f.onHeal).Close(closeCtx, request.GetConnection().Clone(), opts...)
		if err != nil {
			logEntry.Errorf("Failed to close connection %s: %v", request.GetConnection().GetId(), err)
		}
	}
}

// createHealContext - create context to be used on heal.
//                     Uses f.ctx as base and inserts Candidates from requestCtx or cachedCtx into it, if there are any.
func (f *healServer) createHealContext(requestCtx, cachedCtx context.Context) context.Context {
	ctx := requestCtx
	if cachedCtx != nil {
		if candidates := discover.Candidates(ctx); candidates == nil || len(candidates.Endpoints) > 0 {
			ctx = cachedCtx
		}
	}
	healCtx := f.ctx
	if candidates := discover.Candidates(ctx); candidates != nil {
		healCtx = discover.WithCandidates(healCtx, candidates.Endpoints, candidates.NetworkService)
	}

	return healCtx
}
