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

// Package injectclock can be used in testing to inject a mockClock into the context of the chain
package injectclock

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type injectClockClient struct {
	clock.Clock
}

// NewClient - client that injects clk into the context as it passes through the chain element.
func NewClient(clk clock.Clock) networkservice.NetworkServiceClient {
	return &injectClockClient{
		Clock: clk,
	}
}

func (i *injectClockClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	ctx = clock.WithClock(ctx, i.Clock)
	return next.Client(ctx).Request(ctx, request)
}

func (i *injectClockClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx = clock.WithClock(ctx, i.Clock)
	return next.Client(ctx).Close(ctx, conn)
}
