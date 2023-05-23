// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package heal

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"
)

type healClient struct {
	chainCtx              context.Context
	livenessCheck         LivenessCheck
	livenessCheckInterval time.Duration
	livenessCheckTimeout  time.Duration
}

// NewClient - returns a new heal client chain element
func NewClient(chainCtx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	o := &options{
		livenessCheckInterval: livenessCheckInterval,
		livenessCheckTimeout:  livenessCheckTimeout,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &healClient{
		chainCtx:              chainCtx,
		livenessCheck:         o.livenessCheck,
		livenessCheckInterval: o.livenessCheckInterval,
		livenessCheckTimeout:  o.livenessCheckTimeout,
	}
}

func (h *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	// Cancel any existing eventLoop
	cancelEventLoop, loaded := loadAndDeleteFull(ctx)
	if loaded {
		cancelEventLoop()
		logrus.Error("reiogna: healClient cancel old loop")
	} else {
		logrus.Error("reiogna: healClient ctx doesn't have loop")
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		logrus.Error("reiogna: healClient exit on error")
		if loaded && h.livenessCheck != nil {
			logrus.Error("reiogna: healClient start new datapath loop")
			cancelEventLoop, eventLoopErr := newDataPlaneEventLoop(
				extend.WithValuesFromContext(h.chainCtx, ctx), request.Connection, h)
			if eventLoopErr != nil {
				logrus.Error("reiogna: healClient datapath loop error ", eventLoopErr)
				closeCtx, closeCancel := closeCtxFunc()
				defer closeCancel()
				_, _ = next.Client(closeCtx).Close(closeCtx, conn)
				return nil, err
			}
			storeCancelData(ctx, cancelEventLoop)
		}
		return nil, err
	}
	logrus.Error("reiogna: healClient success, reset dp loop")
	cancelDataEventLoop, loaded := loadAndDeleteData(ctx)
	if loaded {
		cancelDataEventLoop()
		logrus.Error("reiogna: healClient cancel data loop")
	} else {
		logrus.Error("reiogna: healClient ctx doesn't have data loop")
	}
	cc, ccLoaded := clientconn.Load(ctx)
	if ccLoaded {
		logrus.Error("reiogna: healClient start new loop")
		cancelEventLoop, eventLoopErr := newEventLoop(
			extend.WithValuesFromContext(h.chainCtx, ctx), cc, conn, h)
		if eventLoopErr != nil {
			closeCtx, closeCancel := closeCtxFunc()
			defer closeCancel()
			_, _ = next.Client(closeCtx).Close(closeCtx, conn)
			return nil, eventLoopErr
		}
		storeCancelFull(ctx, cancelEventLoop)
	}
	return conn, nil
}

func (h *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDeleteFull(ctx); loaded {
		cancelEventLoop()
	}
	return next.Client(ctx).Close(ctx, conn)
}
