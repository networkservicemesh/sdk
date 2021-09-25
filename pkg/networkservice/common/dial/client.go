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
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	// If we already have an old cc, close it
	di, diLoaded := loadAndDelete(ctx)
	if diLoaded && di != nil && di.cc != nil {
		_ = di.cc.Close()
	}

	// If there's already a grpc.ClientConn via other means and it's not ours
	// don't create one
	oldCC, ccLoaded := clientconn.Load(ctx)
	if ccLoaded && di != nil && oldCC != di.cc {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	dialCtx := ctx
	if d.dialTimeout != 0 {
		dialCtx, _ = clock.FromContext(d.chainCtx).WithTimeout(dialCtx, d.dialTimeout)
	}
	cc, err := grpc.DialContext(dialCtx, grpcutils.URLToTarget(clientURL), d.dialOptions...)
	if err != nil {
		if cc != nil {
			_ = cc.Close()
		}
		return nil, errors.Wrapf(err, "failed to dial %s", grpcutils.URLToTarget(clientURL))
	}
	// store the cc because it will be needed by later chain elements like connect
	clientconn.Store(ctx, cc)
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		_ = cc.Close()
		return nil, err
	}
	go func() {
		<-d.chainCtx.Done()
		_ = cc.Close()
	}()
	store(ctx, &dialInfo{
		url: clientURL,
		cc:  cc,
	})
	return conn, nil

}

func (d *dialClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	_, err := next.Client(ctx).Close(ctx, conn, opts...)

	di, diLoaded := loadAndDelete(ctx)
	if !diLoaded {
		return &emptypb.Empty{}, err
	}

	if di != nil && di.cc != nil {
		di.cc.Close()
	}
	clientconn.Delete(ctx)
	return &emptypb.Empty{}, err
}
