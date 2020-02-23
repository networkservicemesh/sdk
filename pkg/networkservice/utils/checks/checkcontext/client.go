// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package checkcontext - provides networkservice chain elements for checking the context.Context passed on by the previous chain element
package checkcontext

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkContextAfterClient struct {
	*testing.T
	check func(*testing.T, context.Context)
}

// NewClient - returns NetworkServiceClient that checks the context passed in from the previous Client in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewClient(t *testing.T, check func(*testing.T, context.Context)) networkservice.NetworkServiceClient {
	return &checkContextAfterClient{
		T:     t,
		check: check,
	}
}

func (t *checkContextAfterClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	t.check(t.T, ctx)
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (t *checkContextAfterClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	t.check(t.T, ctx)
	return next.Client(ctx).Close(ctx, conn, opts...)
}
