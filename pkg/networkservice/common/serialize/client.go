// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package serialize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

type serializeClient struct {
	executor multiexecutor.MultiExecutor
}

// NewClient returns a new serialize client chain element
func NewClient() networkservice.NetworkServiceClient {
	return new(serializeClient)
}

func (c *serializeClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	connID := request.GetConnection().GetId()
	<-c.executor.AsyncExec(connID, func() {
		requestCtx := WithExecutor(ctx, executorFunc(c.executor.Executor(connID)))
		conn, err = next.Client(ctx).Request(requestCtx, request, opts...)
	})
	return conn, err
}

func (c *serializeClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	<-c.executor.AsyncExec(conn.GetId(), func() {
		requestCtx := WithExecutor(ctx, executorFunc(c.executor.Executor(conn.GetId())))
		_, err = next.Client(ctx).Close(requestCtx, conn, opts...)
	})
	return new(empty.Empty), err
}
