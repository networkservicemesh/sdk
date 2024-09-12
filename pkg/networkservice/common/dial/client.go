// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
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
	closeContextFunc := postpone.ContextWithValues(ctx)
	// If no clientURL, we have no work to do
	// call the next in the chain
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	cc, loaded := clientconn.LoadOrStore(ctx, newDialer(d.chainCtx, d.dialTimeout, d.dialOptions...))

	// If there's an existing grpc.ClientConnInterface and it's not ours, call the next in the chain
	di, ok := cc.(*dialer)
	if !ok {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	// If our existing dialer has a different URL close down the chain
	if di.clientURL != nil && di.clientURL.String() != clientURL.String() {
		closeCtx, closeCancel := closeContextFunc()
		defer closeCancel()
		err := di.Dial(closeCtx, di.clientURL)
		if err != nil {
			log.FromContext(ctx).Errorf("can not redial to %v, err %v. Deleting clientconn...", grpcutils.URLToTarget(di.clientURL), err)
			clientconn.Delete(ctx)
			return nil, err
		}
		_, _ = next.Client(ctx).Close(clienturlctx.WithClientURL(closeCtx, di.clientURL), request.GetConnection(), opts...)
	}

	err := di.Dial(ctx, clientURL)
	if err != nil {
		log.FromContext(ctx).Errorf("Can not dial to %v", grpcutils.URLToTarget(clientURL))
		if !loaded {
			log.FromContext(ctx).Errorf("Deleting clientconn...")
			_ = di.Close()
			clientconn.Delete(ctx)
		}
		return nil, err
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		if !loaded {
			_ = di.Close()
			clientconn.Delete(ctx)
		}
		return nil, err
	}
	return conn, nil
}

func (d *dialClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cc, _ := clientconn.Load(ctx)
	di, ok := cc.(*dialer)
	if !ok {
		return next.Client(ctx).Close(ctx, conn, opts...)
	}
	defer func() {
		err := di.Close()
		if err != nil {
			log.FromContext(ctx).Errorf("dialer close failed: %v", err)
		}
		clientconn.Delete(ctx)
	}()

	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		log.FromContext(ctx).Info("closing the connection using the old url")
		clientURL = di.clientURL
	}
	err := di.Dial(ctx, clientURL)
	if err != nil {
		log.FromContext(ctx).Errorf("dial failed: %v", err)
	}

	// Call next regardless of connection state.
	// Even if there is something wrong with connection,
	// we need to clear resources in the current app.
	return next.Client(ctx).Close(ctx, conn, opts...)
}
