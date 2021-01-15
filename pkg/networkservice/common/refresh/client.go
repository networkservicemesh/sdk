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

// Package refresh periodically resends NetworkServiceMesh.Request for an existing connection
// so that the Endpoint doesn't 'expire' the networkservice.
package refresh

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

type refreshClient struct {
	ctx      context.Context
	timers   map[string]*time.Timer        // key == request.GetConnection.GetId()
	cancels  map[string]context.CancelFunc // key == request.GetConnection.GetId()
	executor serialize.Executor
}

// NewClient - creates new NetworkServiceClient chain element for refreshing connections before they timeout at the
// endpoint
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	rv := &refreshClient{
		ctx:     ctx,
		timers:  make(map[string]*time.Timer),
		cancels: make(map[string]context.CancelFunc),
	}
	return rv
}

func (t *refreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	expireTime, err := ptypes.Timestamp(rv.GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetExpires())
	if err != nil {
		return nil, err
	}

	// Create refreshRequest
	refreshRequest := request.Clone()
	refreshRequest.Connection = rv.Clone()

	// TODO - introduce random noise into duration avoid timer lock
	duration := time.Until(expireTime) / 3
	t.executor.AsyncExec(func() {
		// Create the refresh context
		var cancel context.CancelFunc
		refreshCtx, cancel := context.WithCancel(extend.WithValuesFromContext(t.ctx, ctx))
		if deadline, ok := ctx.Deadline(); ok {
			refreshCtx, cancel = context.WithDeadline(refreshCtx, deadline.Add(duration))
		}

		connID := refreshRequest.GetConnection().GetId()

		// Stop any existing timers
		if timer, ok := t.timers[connID]; ok {
			timer.Stop()
		}

		// Set new timer
		var timer *time.Timer
		timer = time.AfterFunc(duration, func() {
			<-t.executor.AsyncExec(func() {
				// Check to see if we've been superseded by another timer, if so, do nothing
				currentTimer, ok := t.timers[connID]
				if ok && currentTimer != timer {
					cancel()
					return
				}
			})
			// Resend request if the refreshCtx is not Done.  Note: refreshCtx should only be done if
			// t.ctx has been canceled (usually because the clientCtx has been canceled
			select {
			case <-refreshCtx.Done():
			default:
				if _, err := t.Request(refreshCtx, refreshRequest, opts...); err != nil {
					// TODO - do we want to retry at 2/3 and 3/3 if we fail here?
					logger.Log(refreshCtx).Errorf("Error while attempting to refresh connection %s: %+v", connID, err)
				}
			}
			// Set timer to nil to be really really sure we don't have a circular reference that precludes garbage collection
			timer = nil
		})
		t.timers[connID] = timer
		t.cancels[connID] = cancel
	})
	return rv, nil
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	t.executor.AsyncExec(func() {
		if cancel, ok := t.cancels[conn.GetId()]; ok {
			cancel()
			delete(t.cancels, conn.GetId())
		}
		if timer, ok := t.timers[conn.GetId()]; ok {
			timer.Stop()
			delete(t.timers, conn.GetId())
		}
	})
	return next.Client(ctx).Close(ctx, conn)
}
