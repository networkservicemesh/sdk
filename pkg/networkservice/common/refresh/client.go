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

// Package refresh periodically resends NetworkServiceMesh.Request for an existing connection
// so that the Endpoint doesn't 'expire' the networkservice.
package refresh

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type refreshClient struct {
	connectionTimers  map[string]*time.Timer
	refreshCancellers map[string]func()
	executor          serialize.Executor
}

// NewClient - creates new NetworkServiceClient chain element for refreshing connections before they timeout at the
// endpoint
func NewClient() networkservice.NetworkServiceClient {
	rv := &refreshClient{
		connectionTimers:  make(map[string]*time.Timer),
		refreshCancellers: make(map[string]func()),
	}
	return rv
}

func (t *refreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	rv, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	// Clone the request
	req := request.Clone()
	// Set its connection to the returned connection we received
	req.Connection = rv
	// Setup the timer with the req containing the returned connection

	expire, err := t.getExpireDuration(request)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating timer from Request.Connection.Path.PathSegment[%d].ExpireTime", request.GetConnection().GetPath().GetIndex())
	}
	t.executor.AsyncExec(func() {
		if refreshCtx := refreshContext(ctx); refreshCtx != nil && refreshCtx.Err() != nil {
			return
		}
		if timer, ok := t.connectionTimers[req.GetConnection().GetId()]; ok {
			timer.Stop()
		}
		if canceller, ok := t.refreshCancellers[req.GetConnection().GetId()]; ok {
			canceller()
		}
		timer, cancelFunc := t.createTimer(ctx, req, expire, opts...)
		t.connectionTimers[req.GetConnection().GetId()] = timer
		t.refreshCancellers[req.GetConnection().GetId()] = cancelFunc
	})
	return rv, nil
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	t.executor.AsyncExec(func() {
		if timer, ok := t.connectionTimers[conn.GetId()]; ok {
			timer.Stop()
			delete(t.connectionTimers, conn.GetId())
		}
		if canceller, ok := t.refreshCancellers[conn.GetId()]; ok {
			canceller()
			delete(t.refreshCancellers, conn.GetId())
		}
	})
	return next.Client(ctx).Close(ctx, conn)
}

func (t *refreshClient) getExpireDuration(request *networkservice.NetworkServiceRequest) (time.Duration, error) {
	expireTime, err := ptypes.Timestamp(request.GetConnection().GetPath().GetPathSegments()[request.GetConnection().GetPath().GetIndex()].GetExpires())
	if err != nil {
		return 0, err
	}
	duration := time.Until(expireTime)
	return duration, nil
}

func (t *refreshClient) createTimer(ctx context.Context, request *networkservice.NetworkServiceRequest, expires time.Duration, opts ...grpc.CallOption) (timer *time.Timer, cancelFunc func()) {
	refreshCtx, cancelFunc := context.WithCancel(context.Background())
	newCtx := withRefreshContext(extend.WithValuesFromContext(context.Background(), ctx), refreshCtx)

	timer = time.AfterFunc(expires, func() {
		// TODO what to do about error handling?
		// TODO what to do about expiration of context
		if _, err := t.Request(newCtx, request, opts...); err != nil {
			trace.Log(newCtx).Errorf("Error while attempting to refresh connection %s: %+v", request.GetConnection().GetId(), err)
		}
	})
	return timer, cancelFunc
}
