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

// Package dial will dial up a grpc.ClientConnInterface if a client *url.URL is provided in the ctx, retrievable by
// clienturlctx.ClientURL(ctx) and put the resulting grpc.ClientConnInterface into the ctx using clientconn.Store(..)
// where it can be retrieved by other chain elements using clientconn.Load(...)
package dial

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type dialClient struct {
	chainCtx    context.Context
	dialOptions []grpc.DialOption
	dialTimeout time.Duration
}

// NewClient - returns new dial chain element
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	o := &option{}
	for _, opt := range opts {
		opt(o)
	}
	return &dialClient{
		chainCtx:    chainCtx,
		dialOptions: o.dialOptions,
		dialTimeout: o.dialTimeout,
	}
}

func (d *dialClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// If no clientURL, we have no work to do
	// call the next in the chain
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	di, err := d.dial(ctx, func(di *dialInfo) {
		_, _ = d.Close(clienturlctx.WithClientURL(ctx, di.url), request.GetConnection(), opts...)
	})
	if di != nil {
		store(ctx, di)
	}
	if err != nil {
		return nil, err
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		_ = di.Close()
		return nil, err
	}
	return conn, nil
}

func (d *dialClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// If no clientURL, we have no work to do
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}

	di, _ := d.dial(ctx, func(_ *dialInfo) {})

	_, err := next.Client(ctx).Close(ctx, conn, opts...)
	if di != nil {
		_ = di.Close()
	}
	clientconn.Delete(ctx)
	return &emptypb.Empty{}, err
}

func (d *dialClient) dial(ctx context.Context, onRedirect func(di *dialInfo)) (di *dialInfo, err error) {
	clientURL := clienturlctx.ClientURL(ctx)

	// If we already have an old cc, close it
	di, diLoaded := loadAndDelete(ctx)
	if diLoaded {
		_ = di.Close()
	}

	if di != nil && di.url.String() != clientURL.String() {
		onRedirect(di)
	}

	// Setup dialTimeout if needed
	dialCtx := ctx
	if d.dialTimeout != 0 {
		dialCtx, _ = clock.FromContext(d.chainCtx).WithTimeout(dialCtx, d.dialTimeout)
	}

	// Dial
	target := grpcutils.URLToTarget(clientURL)
	cc, err := grpc.DialContext(dialCtx, target, d.dialOptions...)
	if err != nil {
		if cc != nil {
			_ = cc.Close()
		}
		return di, errors.Wrapf(err, "failed to dial %s", target)
	}

	// store the cc because it will be needed by later chain elements like connect
	clientconn.Store(ctx, cc)
	go func() {
		<-d.chainCtx.Done()
		_ = cc.Close()
	}()
	return &dialInfo{
		url:        clientURL,
		ClientConn: cc,
	}, nil
}
