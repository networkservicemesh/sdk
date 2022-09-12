// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// Package injectopts - injects grpc.CallOptions by appending them to the end of the opts... it receives before calling
//
//	the next client in the chain
package injectopts

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type injectOptsClient struct {
	opts []grpc.CallOption
}

// NewClient - injects opts grpc.CallOptions by appending them to the end of the opts... it receives before calling
//
//	the next client in the chain
func NewClient(opts ...grpc.CallOption) networkservice.NetworkServiceClient {
	if len(opts) == 0 {
		opts = append(opts, grpc.EmptyCallOption{})
	}
	return &injectOptsClient{opts: opts}
}

func (i *injectOptsClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	opts = append(opts, i.opts...)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (i *injectOptsClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	opts = append(opts, i.opts...)
	return next.Client(ctx).Close(ctx, conn, opts...)
}
