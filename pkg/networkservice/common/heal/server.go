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

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ctxWrapper struct {
	mut            sync.Mutex
	request        *networkservice.NetworkServiceRequest
	requestTimeout time.Duration
	ctx            context.Context
	cancel         func()
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
	clockTime := clock.FromContext(ctx)
	ctx = f.withHandlers(ctx)

	requestTimeout := time.Duration(0)
	if deadline, ok := ctx.Deadline(); ok {
		requestTimeout = clockTime.Until(deadline)
	}

	requestStart := clockTime.Now()

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// There can possible be a case when we are trying to heal from the local case to the remote case. Maximum captured
	// difference between these times was 3x on packet cluster (0.5s local vs 1.5s remote). So taking 5x value would be
	// enough to cover such local to remote case and not too much in terms of blocking subsequent Request/Close events
	// (7.5s for the worst remote case).
	requestDuration := clockTime.Since(requestStart) * 5
	if requestDuration > requestTimeout {
		requestTimeout = requestDuration
	}

	cw, loaded := f.healContextMap.LoadOrStore(request.GetConnection().GetId(), &ctxWrapper{
		request:        request.Clone(),
		requestTimeout: requestTimeout,
		ctx:            f.createHealContext(ctx, nil),
	})
	if loaded {
		cw.mut.Lock()
		defer cw.mut.Unlock()

		if cw.cancel != nil {
			log.FromContext(ctx).Debug("canceling previous heal")
			cw.cancel()
			cw.cancel = nil
		}
		cw.request = request.Clone()
		if requestTimeout > cw.requestTimeout {
			cw.requestTimeout = requestTimeout
		}
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

func (f *healServer) getHealContext(
	conn *networkservice.Connection,
) (context.Context, *networkservice.NetworkServiceRequest, time.Duration) {
	cw, ok := f.healContextMap.Load(conn.GetId())
	if !ok {
		return nil, nil, 0
	}

	cw.mut.Lock()
	defer cw.mut.Unlock()

	if cw.cancel != nil {
		log.FromContext(cw.ctx).Debug("canceling previous heal")
		cw.cancel()
	}
	ctx, cancel := context.WithCancel(cw.ctx)
	cw.cancel = cancel

	return ctx, cw.request.Clone(), cw.requestTimeout
}

// handleHealConnectionRequest - heals requested connection. Returns immediately, heal is asynchronous.
func (f *healServer) handleHealConnectionRequest(conn *networkservice.Connection) {
	ctx, request, requestTimeout := f.getHealContext(conn)
	if request == nil {
		return
	}

	request.SetRequestConnection(conn.Clone())

	go f.processHeal(ctx, request, requestTimeout)
}

// handleRestoreConnectionRequest - recreates connection. Returns immediately, heal is asynchronous.
func (f *healServer) handleRestoreConnectionRequest(conn *networkservice.Connection) {
	ctx, request, requestTimeout := f.getHealContext(conn)
	if request == nil {
		return
	}

	request.SetRequestConnection(conn.Clone())

	go f.restoreConnection(ctx, request, requestTimeout)
}

func (f *healServer) stopHeal(conn *networkservice.Connection) {
	cw, loaded := f.healContextMap.LoadAndDelete(conn.GetId())
	if !loaded {
		return
	}

	cw.mut.Lock()
	defer cw.mut.Unlock()

	if cw.cancel != nil {
		log.FromContext(cw.ctx).Debug("canceling previous heal")
		cw.cancel()
	}
}

func (f *healServer) restoreConnection(
	ctx context.Context,
	request *networkservice.NetworkServiceRequest,
	requestTimeout time.Duration,
) {
	clockTime := clock.FromContext(ctx)

	if ctx.Err() != nil {
		return
	}

	// Make sure we have a valid expireTime to work with
	expires := request.GetConnection().GetNextPathSegment().GetExpires()
	expireTime, err := ptypes.Timestamp(expires)
	if err != nil {
		return
	}

	deadline := clockTime.Now().Add(f.restoreTimeout)
	if deadline.After(expireTime) {
		deadline = expireTime
	}
	restoreCtx, restoreCancel := clockTime.WithDeadline(ctx, deadline)
	defer restoreCancel()

	for restoreCtx.Err() == nil {
		requestCtx, requestCancel := clockTime.WithTimeout(restoreCtx, requestTimeout)
		_, err := (*f.onHeal).Request(requestCtx, request.Clone())
		requestCancel()

		if err == nil {
			return
		}
	}

	f.processHeal(ctx, request, requestTimeout)
}

func (f *healServer) processHeal(
	ctx context.Context,
	request *networkservice.NetworkServiceRequest,
	requestTimeout time.Duration,
) {
	clockTime := clock.FromContext(ctx)

	fields := make(map[string]interface{})
	if prevFields := log.Fields(ctx); prevFields != nil {
		for k, v := range prevFields {
			fields[k] = v
		}
	}
	fields["healServer"] = "processHeal"
	ctx = log.WithFields(ctx, fields)
	logger := log.FromContext(ctx)

	if ctx.Err() != nil {
		return
	}

	candidates := discover.Candidates(ctx)
	conn := request.GetConnection()
	if candidates != nil || conn.GetPath().GetIndex() == 0 {
		logger.Infof("Starting heal process for %s", conn.GetId())

		conn.NetworkServiceEndpointName = ""
		conn.Path.PathSegments = conn.Path.PathSegments[0 : conn.Path.Index+1]

		for ctx.Err() == nil {
			requestCtx, requestCancel := clockTime.WithTimeout(ctx, requestTimeout)
			_, err := (*f.onHeal).Request(requestCtx, request.Clone())
			requestCancel()

			if err != nil {
				logger.Errorf("Failed to heal connection %s: %v", conn.GetId(), err)
			} else {
				logger.Infof("Finished heal process for %s", conn.GetId())
				break
			}
		}
	} else {
		logger.Warn("closing the connection")
		// Huge timeout is not required to close connection on a current path segment
		closeCtx, closeCancel := clockTime.WithTimeout(ctx, time.Second)
		defer closeCancel()

		_, err := (*f.onHeal).Close(closeCtx, conn)
		if err != nil {
			logger.Errorf("Failed to close connection %s: %v", conn.GetId(), err)
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
	healCtx = log.WithFields(healCtx, log.Fields(requestCtx))

	return healCtx
}
