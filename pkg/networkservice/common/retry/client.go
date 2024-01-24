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

// Package retry provides a networkservice.NetworksrviceClient wrapper that allows to retries requests and closes.
package retry

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type retryClient struct {
	interval   time.Duration
	tryTimeout time.Duration
	client     networkservice.NetworkServiceClient
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

// NewClient - returns a connect chain element
func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	var result = &retryClient{
		interval:   time.Millisecond * 200,
		tryTimeout: time.Second * 15,
	}

	for _, opt := range opts {
		opt(result)
	}

	return result
}

func (r *retryClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	logger := log.FromContext(ctx).WithField("retryClient", "Request")
	c := clock.FromContext(ctx)

	for ctx.Err() == nil {
		requestCtx, cancel := c.WithTimeout(ctx, r.tryTimeout)
		resp, err := next.Client(ctx).Request(requestCtx, request.Clone(), opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.After(r.interval):
				continue
			}
		}

		return resp, err
	}

	return nil, ctx.Err()
}

func (r *retryClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn.Clone(), opts...)
}
