// Copyright (c) 2022 Cisco and/or its affiliates.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package checkresponse - provides networkservice chain elements to check the response received from the next element in the chain
package checkresponse

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkResponseClient struct {
	*testing.T
	check func(*testing.T, *networkservice.Connection)
}

// NewClient - returns NetworkServiceClient chain elements to check the response received from the next element in the chain
//
//	t - *testing.T for checks
//	check - function to check the Connnection
func NewClient(t *testing.T, check func(*testing.T, *networkservice.Connection)) networkservice.NetworkServiceClient {
	return &checkResponseClient{
		T:     t,
		check: check,
	}
}

func (c *checkResponseClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	resp, err := next.Client(ctx).Request(ctx, request, opts...)
	c.check(c.T, resp)
	return resp, err
}

func (c *checkResponseClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
