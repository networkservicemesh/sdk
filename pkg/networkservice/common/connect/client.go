// Copyright (c) 2021 Cisco and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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

package connect

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/dial"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	defaultDialTimeout = time.Millisecond * 100
	initialTimeout     = time.Millisecond * 20
	timeoutMultiplier  = 2
)

type option struct {
	dialTimeout time.Duration
}

// Option - options for the dial chain element
type Option func(*option)

// WithDialTimeout - dialTimeout for use by dial chain element.
func WithDialTimeout(dialTimeout time.Duration) Option {
	return func(o *option) {
		o.dialTimeout = dialTimeout
	}
}

type connectClient struct {
	dialTimeout time.Duration
}

// NewClient - returns a connect chain element
func NewClient(opts ...Option) networkservice.NetworkServiceClient {
	o := &option{
		dialTimeout: defaultDialTimeout,
	}
	for _, opt := range opts {
		opt(o)
	}

	if o.dialTimeout == time.Duration(0) {
		panic("dial timeout in connectClient can't be equal to zero")
	}
	return &connectClient{
		dialTimeout: o.dialTimeout,
	}
}

func (c *connectClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errors.New("no grpc.ClientConnInterface provided")
	}

	if dialer, ok := cc.(*dial.Dialer); ok {
		grpcClientConn := dialer.ClientConn

		ready := waitForReady(ctx, grpcClientConn, c.dialTimeout)
		if !ready {
			return nil, errors.New("dial error")
		}
	}

	return networkservice.NewNetworkServiceClient(cc).Request(ctx, request, opts...)
}

func (c *connectClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errors.New("no grpc.ClientConnInterface provided")
	}
	return networkservice.NewNetworkServiceClient(cc).Close(ctx, conn, opts...)
}

func waitForReady(ctx context.Context, conn *grpc.ClientConn, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	currentTimeout := initialTimeout

	for time.Now().Add(currentTimeout).Before(deadline) {
		after := time.After(currentTimeout)
		select {
		case <-after:
			if conn.GetState() == connectivity.Ready || conn.GetState() == connectivity.Idle {
				return true
			}
		case <-ctx.Done():
			return false
		}
		currentTimeout *= timeoutMultiplier
		if time.Now().Add(currentTimeout).After(deadline) {
			currentTimeout = time.Until(deadline)
		}
		log.FromContext(ctx).Infof("currentTimeout: %v", currentTimeout.Milliseconds())
	}
	return false
}
