// Copyright (c) 2024 Cisco and/or its affiliates.
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

// Package limit provides a chain element that can set limits for the RPC calls.
package limit

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type Option func(c *limitClient)

// WithDialLimit sets dial limit
func WithDialLimit(d time.Duration) Option {
	return func(c *limitClient) {
		c.dialLimit = d
	}
}

type limitClient struct {
	dialLimit time.Duration
}

func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	ret := &limitClient{
		dialLimit: time.Minute,
	}

	for _, opt := range opts {
		opt(ret)
	}

	return ret
}

func (n *limitClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.Server(ctx).Request(ctx, request)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.Server(ctx).Request(ctx, request)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleClient", "Request")

	go func() {
		select {
		case <-time.After(n.dialLimit):
			logger.Warn("Reached dial limit, closing connection...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (n *limitClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.Server(ctx).Close(ctx, conn)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.Server(ctx).Close(ctx, conn)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleClient", "Close")

	go func() {
		select {
		case <-time.After(n.dialLimit):
			logger.Warn("Reached dial limit, closing connection...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()
	return next.Client(ctx).Close(ctx, conn, opts...)
}
