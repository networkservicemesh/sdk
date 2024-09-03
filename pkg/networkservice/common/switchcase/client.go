// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package switchcase

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// ClientCase is a case type for the switch-case client chain element.
type ClientCase struct {
	Condition Condition
	Client    networkservice.NetworkServiceClient
}

type switchClient struct {
	cases []*ClientCase
}

// NewClient returns a new switch-case client chain element.
func NewClient(cases ...*ClientCase) networkservice.NetworkServiceClient {
	return &switchClient{
		cases: cases,
	}
}

func (s *switchClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	for _, c := range s.cases {
		if c.Condition(ctx, request.GetConnection()) {
			return c.Client.Request(ctx, request, opts...)
		}
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (s *switchClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	for _, c := range s.cases {
		if c.Condition(ctx, conn) {
			return c.Client.Close(ctx, conn, opts...)
		}
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
