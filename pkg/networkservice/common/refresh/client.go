// Copyright (c) 2020-2024 Cisco Systems, Inc.
//
// Copyright (c) 2020-2024 Doc.ai and/or its affiliates.
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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
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
	logger := log.FromContext(ctx).WithField("refreshClient", "Request")

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	// Compute refreshAfter
	refreshAfter := after(ctx, conn)

	// Create a cancel context.
	cancelCtx, cancel := context.WithCancel(t.chainCtx)

	if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(t)); loaded {
		oldCancel()
	}
	store(ctx, metadata.IsClient(t), cancel)

	eventFactory := begin.FromContext(ctx)
	// Create the afterCh *outside* the go routine.  This must be done to avoid picking up a later 'now'
	// from mockClock in testing
	afterCh := clock.FromContext(ctx).After(refreshAfter)
	go func() {
		for {
			select {
			case <-cancelCtx.Done():
				return
			case <-afterCh:
				if err := <-eventFactory.Request(begin.CancelContext(cancelCtx)); err != nil {
					afterCh = clock.FromContext(ctx).After(time.Millisecond * 200)
					logger.Warnf("refresh failed: %s", err.Error())
					continue
				}
				return
			}
		}
	}()

	return conn, nil
}

func (t *refreshClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (e *empty.Empty, err error) {
	if oldCancel, loaded := loadAndDelete(ctx, metadata.IsClient(t)); loaded {
		oldCancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}

func after(ctx context.Context, conn *networkservice.Connection) time.Duration {
	clockTime := clock.FromContext(ctx)

	var minTimeout *time.Duration
	var expireTime time.Time
	for _, segment := range conn.GetPath().GetPathSegments() {
		expTime := segment.GetExpires().AsTime()

		timeout := clockTime.Until(expTime)

		if minTimeout == nil || timeout < *minTimeout {
			if minTimeout == nil {
				minTimeout = new(time.Duration)
			}

			*minTimeout = timeout
			expireTime = expTime
		}
	}

	if minTimeout != nil {
		log.FromContext(ctx).Infof("expiration after %s at %s", minTimeout.String(), expireTime.UTC())
	}

	if minTimeout == nil || *minTimeout <= 0 {
		return 1
	}

	// A heuristic to reduce the number of redundant requests in a chain
	// made of refreshing clients with the same expiration time: let outer
	// chain elements refresh slightly faster than inner ones.
	// Update interval is within 0.2*expirationTime .. 0.4*expirationTime
	scale := 1. / 3.
	path := conn.GetPath()
	if len(path.GetPathSegments()) > 1 {
		scale = 0.2 + 0.2*float64(path.GetIndex())/float64(len(path.GetPathSegments()))
	}
	duration := time.Duration(float64(*minTimeout) * scale)

	return duration
}
