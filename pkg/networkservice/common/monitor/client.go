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

package monitor

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type monitorClient struct {
	chainCtx context.Context
}

// NewClient - New client to monitor server and update EventConsumers
func NewClient(chainCtx context.Context) networkservice.NetworkServiceClient {
	return &monitorClient{
		chainCtx,
	}
}

func (m *monitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	// Cancel any existing clientEventLoop
	if cancelClientEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelClientEventLoop()
	}
	// Call downstream
	conn, err := next.Client(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Start a new clientEventLoop
	cc, ccLoaded := clientconn.Load(ctx)
	if ccLoaded {
		cancelClientEventLoop, eventLoopErr := newClientEventLoop(m.chainCtx, FromContext(ctx), cc, conn)
		if eventLoopErr != nil {
			return nil, errors.Wrap(eventLoopErr, "unable to monitor")
		}
		store(ctx, metadata.IsClient(m), cancelClientEventLoop)
	}

	return conn, err
}

func (m *monitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	// Cancel any existing clientEventLoop
	if cancelClientEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelClientEventLoop()
	}
	return next.Client(ctx).Close(ctx, conn)
}
