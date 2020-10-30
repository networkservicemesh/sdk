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

package tracehelper

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

// traceHelperClient NetworkServiceServer, that injects ConnectionInfo into context
type traceHelperClient struct {
}

// NewClient - creates a new traceHelper client to inject ConnectionInfo into context.
func NewClient() networkservice.NetworkServiceClient {
	return &traceHelperClient{}
}

func (th *traceHelperClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	ctx = withConnectionInfo(ctx, &ConnectionInfo{})
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (th *traceHelperClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = withConnectionInfo(ctx, &ConnectionInfo{})
	return next.Client(ctx).Close(ctx, conn, opts...)
}
