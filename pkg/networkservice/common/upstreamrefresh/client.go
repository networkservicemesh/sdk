// Copyright (c) 2022 Cisco and/or its affiliates.
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

package upstreamrefresh

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type upstreamRefreshClient struct {
	chainCtx      context.Context
	localNotifier *notifier
}

// NewClient - returns a new upstreamrefresh chain element.
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}

	return &upstreamRefreshClient{
		chainCtx:      chainCtx,
		localNotifier: o.localNotifier,
	}
}

func (u *upstreamRefreshClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx); loaded {
		cancelEventLoop()
	}
	u.localNotifier.unsubscribe(request.GetConnection().GetId())

	if u.localNotifier != nil {
		storeLocalNotifier(ctx, metadata.IsClient(u), u.localNotifier)
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	u.localNotifier.subscribe(conn.GetId())

	cc, ccLoaded := clientconn.Load(ctx)
	if ccLoaded {
		cancelEventLoop, eventLoopErr := newEventLoop(
			extend.WithValuesFromContext(u.chainCtx, ctx), cc, conn, u.localNotifier)
		if eventLoopErr != nil {
			closeCtx, closeCancel := closeCtxFunc()
			defer closeCancel()
			_, _ = next.Client(closeCtx).Close(closeCtx, conn)
			return nil, errors.Wrap(eventLoopErr, "unable to monitor")
		}
		store(ctx, cancelEventLoop)
	}
	return conn, nil
}

func (u *upstreamRefreshClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// Unsubscribe from local notifications
	u.localNotifier.unsubscribe(conn.GetId())

	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx); loaded {
		cancelEventLoop()
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}
