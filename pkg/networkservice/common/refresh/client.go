// Copyright (c) 2020 Cisco Systems, Inc.
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package refresh periodically resends NetworkServiceMesh.Request for an
// existing connection so that the Endpoint doesn't 'expire' the networkservice.
package refresh

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type refreshClient struct {
	chainCtx context.Context
	timers   timerMap
}

// NewClient - creates new NetworkServiceClient chain element for refreshing
// connections before they timeout at the endpoint.
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	return &refreshClient{
		chainCtx: ctx,
	}
}

func (t *refreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	connectionID := request.Connection.Id
	t.stopTimer(connectionID)

	rv, err := next.Client(ctx).Request(ctx, request, opts...)

	executor := serializectx.GetExecutor(ctx, connectionID)
	if executor == nil {
		return nil, errors.New("no executor provided")
	}
	request.Connection = rv.Clone()
	t.startTimer(ctx, connectionID, request, opts)

	return rv, err
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (e *empty.Empty, err error) {
	t.stopTimer(conn.Id)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (t *refreshClient) stopTimer(connectionID string) {
	value, loaded := t.timers.LoadAndDelete(connectionID)
	if loaded {
		value.Stop()
	}
}

func (t *refreshClient) startTimer(ctx context.Context, connectionID string, request *networkservice.NetworkServiceRequest, opts []grpc.CallOption) {
	clockTime := clock.FromContext(ctx)

	nextClient := next.Client(ctx)
	if !request.GetConnection().GetCurrentPathSegment().GetExpires().IsValid() {
		return
	}
	expireTime := request.GetConnection().GetCurrentPathSegment().GetExpires().AsTime()

	// A heuristic to reduce the number of redundant requests in a chain
	// made of refreshing clients with the same expiration time: let outer
	// chain elements refresh slightly faster than inner ones.
	// Update interval is within 0.2*expirationTime .. 0.4*expirationTime
	scale := 1. / 3.
	path := request.GetConnection().GetPath()
	if len(path.PathSegments) > 1 {
		scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
	}
	duration := time.Duration(float64(clockTime.Until(expireTime)) * scale)
	req := request.Clone()
	exec := serializectx.GetExecutor(ctx, connectionID)

	var timer clock.Timer
	timer = clockTime.AfterFunc(duration, func() {
		exec.AsyncExec(func() {
			oldTimer, ok := t.timers.Load(connectionID)
			if !ok || oldTimer != timer {
				return
			}

			t.timers.Delete(connectionID)

			// Context is canceled or deadlined.
			if t.chainCtx.Err() != nil {
				return
			}

			timeout := defaultRefreshRequestTimeout
			if timeout > duration {
				timeout = duration
			}
			refreshCtx, cancel := clockTime.WithTimeout(extend.WithValuesFromContext(t.chainCtx, ctx), timeout)
			defer cancel()
			rv, err := nextClient.Request(refreshCtx, req, opts...)

			if err == nil && rv != nil {
				req.Connection = rv
			}

			t.startTimer(ctx, connectionID, req, opts)
		})
	})

	t.stopTimer(connectionID)
	t.timers.Store(connectionID, timer)
}
