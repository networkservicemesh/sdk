// Copyright (c) 2020 Cisco Systems, Inc.
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type refreshClient struct {
	ctx    context.Context
	timers sync.Map
}

// NewClient - creates new NetworkServiceClient chain element for refreshing
// connections before they timeout at the endpoint.
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	return &refreshClient{
		ctx:    ctx,
		timers: sync.Map{},
	}
}

func (t *refreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	connectionID := request.Connection.Id
	t.stopTimer(connectionID)

	rv, err := next.Client(ctx).Request(ctx, request.Clone(), opts...)

	executor := serialize.GetExecutor(ctx)
	if executor == nil {
		return nil, errors.New("no executor provided")
	}
	request.Connection = rv.Clone()
	nextClient := next.Client(ctx)
	t.startTimer(connectionID, executor, nextClient, request, opts)

	return rv, err
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (e *empty.Empty, err error) {
	t.stopTimer(conn.Id)
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (t *refreshClient) stopTimer(connectionID string) {
	value, loaded := t.timers.LoadAndDelete(connectionID)
	if loaded {
		value.(*time.Timer).Stop()
	}
}

func (t *refreshClient) startTimer(connectionID string, exec serialize.Executor, nextClient networkservice.NetworkServiceClient, request *networkservice.NetworkServiceRequest, opts []grpc.CallOption) {
	path := request.GetConnection().GetPath()
	if path == nil || path.PathSegments == nil || len(path.PathSegments) == 0 ||
		path.Index >= uint32(len(path.PathSegments)) {
		return
	}
	expireTime, err := ptypes.Timestamp(path.PathSegments[path.Index].GetExpires())
	if err != nil {
		return
	}

	// A heuristic to reduce the number of redundant requests in a chain
	// made of refreshing clients with the same expiration time: let outer
	// chain elements refresh slightly faster than inner ones.
	// Update interval is within 0.2*expirationTime .. 0.4*expirationTime
	scale := 1. / 3.
	if len(path.PathSegments) > 1 {
		scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
	}
	duration := time.Duration(float64(time.Until(expireTime)) * scale)

	var timer *time.Timer
	timer = time.AfterFunc(duration, func() {
		exec.AsyncExec(func() {
			oldTimer, ok := t.timers.Load(connectionID)
			if !ok || oldTimer.(*time.Timer) != timer {
				return
			}

			t.timers.Delete(connectionID)

			// Context is canceled or deadlined.
			if t.ctx.Err() != nil {
				return
			}

			rv, err := nextClient.Request(t.ctx, request.Clone(), opts...)

			if err == nil && rv != nil {
				request.Connection = rv
			}

			t.startTimer(connectionID, exec, nextClient, request, opts)
		})
	})

	t.timers.Store(connectionID, timer)
}
