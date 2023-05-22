// Copyright (c) 2021 Cisco and/or its affiliates.
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

// Package retry provides a chain element
// that will repeatedly call Request down the chain
// until it succeeds, or until chain or request context is cancelled.
package retry2

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type retryClient struct {
	chainCtx   context.Context
	interval   time.Duration
	tryTimeout time.Duration
}

// Option configuress retry.Client instance.
type Option func(*retryClient)

// WithTryTimeout sets timeout for the request and close operations try.
func WithTryTimeout(tryTimeout time.Duration) Option {
	return func(rc *retryClient) {
		rc.tryTimeout = tryTimeout
	}
}

// WithInterval sets delay interval before next try.
func WithInterval(interval time.Duration) Option {
	return func(rc *retryClient) {
		rc.interval = interval
	}
}

// NewClient - returns a retry client chain element.
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	var result = &retryClient{
		chainCtx:   chainCtx,
		interval:   time.Millisecond * 200,
		tryTimeout: time.Second * 15,
	}

	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (r *retryClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("retry2Client", "Request")
	c := clock.FromContext(ctx)

	for ctx.Err() == nil {
		logrus.Error("reiogna: retry2Client attempt")
		requestCtx, cancel := c.WithTimeout(ctx, r.tryTimeout)
		conn, err := next.Client(ctx).Request(requestCtx, request.Clone(), opts...)
		// conn, err := next.Client(ctx).Request(ctx, request, opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())
			// closeCtx, cancel := c.WithTimeout(ctx, r.tryTimeout)
			// _, _ = next.Client(ctx).Close(closeCtx, conn.Clone(), opts...)
			// cancel()

			logrus.Error("reiogna: retry2Client waiting on timer")
			select {
			case <-r.chainCtx.Done():
				logrus.Error("reiogna: retry2Client chainCtx cancel")
				return nil, r.chainCtx.Err()
			case <-ctx.Done():
				logrus.Error("reiogna: retry2Client cancel")
				return nil, ctx.Err()
			case <-c.After(r.interval):
				logrus.Error("reiogna: retry2Client react on timer")
				if begin.IsRetryAsync(ctx) {
					logrus.Error("reiogna: retry2Client async")
					ev := begin.FromContext(ctx)
					if ev != nil {
						ev.Request()
					}
					return nil, err
				} else {
					logrus.Error("reiogna: retry2Client sync")
				}
				return nil, err
			}
		}

		return conn, err
	}

	return nil, ctx.Err()
}

func (r *retryClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
