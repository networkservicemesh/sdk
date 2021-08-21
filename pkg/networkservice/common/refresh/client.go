// Copyright (c) 2020 Cisco Systems, Inc.
//
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
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type refreshClient struct {
	chainCtx context.Context
}

// NewClient - creates new NetworkServiceClient chain element for refreshing
// connections before they timeout at the endpoint.
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	return &refreshClient{
		chainCtx: ctx,
	}
}

func (t *refreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// Compute refreshAfter
	refreshAfter, err := after(ctx, conn)
	if err != nil {
		// If we can't refresh, we should close down chain
		_, _ = t.Close(ctx, conn)
		return nil, err
	}

	// Create a cancel context.
	cancelCtx, cancel := context.WithCancel(t.chainCtx)
	for {
		// Call the old cancel to cancel any existing refreshes hanging out waiting to go
		if oldCancel, loaded := LoadAndDelete(ctx, metadata.IsClient(t)); loaded {
			oldCancel()
		}
		// Store the cancel context and break out of the loop
		if _, loaded := LoadOrStore(ctx, metadata.IsClient(t), cancel); !loaded {
			break
		}
	}

	eventFactory := begin.FromContext(ctx)
	timeClock := clock.FromContext(ctx)
	// Create the afterCh *outside* the go routine.  This must be done to avoid picking up a later 'now'
	// from mockClock in testing
	afterCh := timeClock.After(refreshAfter)
	go func(cancelCtx context.Context, afterCh <-chan time.Time) {
		select {
		case <-cancelCtx.Done():
		case <-afterCh:
			eventFactory.Request(begin.CancelContext(cancelCtx))
		}
	}(cancelCtx, afterCh)

	return conn, nil
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (e *empty.Empty, err error) {
	if oldCancel, loaded := LoadAndDelete(ctx, metadata.IsClient(t)); loaded {
		oldCancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func after(ctx context.Context, conn *networkservice.Connection) (time.Duration, error) {
	clockTime := clock.FromContext(ctx)
	expireTime, err := ptypes.Timestamp(conn.GetCurrentPathSegment().GetExpires())
	if err != nil {
		return 0, errors.WithStack(err)
	}
	log.FromContext(ctx).Infof("expireTime %s now %s", expireTime, clockTime.Now().UTC())

	// A heuristic to reduce the number of redundant requests in a chain
	// made of refreshing clients with the same expiration time: let outer
	// chain elements refresh slightly faster than inner ones.
	// Update interval is within 0.2*expirationTime .. 0.4*expirationTime
	scale := 1. / 3.
	path := conn.GetPath()
	if len(path.PathSegments) > 1 {
		scale = 0.2 + 0.2*float64(path.Index)/float64(len(path.PathSegments))
	}
	duration := time.Duration(float64(clockTime.Until(expireTime)) * scale)
	return duration, nil
}
