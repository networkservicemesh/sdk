// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package chaincontext contains chain element for setting chain scoped context.Context
package chaincontext

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type chainContextClient struct {
	chainContext context.Context
}

// NewClient creates a new networkservice.NetworkServiceClient chain element that sets pre-chain context.Context
//             - chainContext - context.Context that has lifetime equal to chain's lifetime
func NewClient(chainContext context.Context) networkservice.NetworkServiceClient {
	return &chainContextClient{
		chainContext: chainContext,
	}
}

func (c *chainContextClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	ctx = WithChainContext(ctx, c.chainContext)
	return next.Client(ctx).Request(ctx, in, opts...)
}

func (c *chainContextClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, in, opts...)
}
