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

package connect

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type connectServer struct {
	client      networkservice.NetworkServiceClient
	callOptions []grpc.CallOption
}

// NewServer - returns a connect chain element
func NewServer(client networkservice.NetworkServiceClient, callOptions ...grpc.CallOption) networkservice.NetworkServiceServer {
	return &connectServer{
		client:      client,
		callOptions: callOptions,
	}
}

func (c *connectServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	clientConn, clientErr := c.client.Request(ctx, request, c.callOptions...)
	if clientErr != nil {
		return nil, clientErr
	}
	request.Connection = clientConn
	serverConn, serverErr := next.Server(ctx).Request(ctx, request)
	if serverErr != nil {
		closeCtx, closeCancel := closeCtxFunc()
		defer closeCancel()
		_, _ = c.client.Close(closeCtx, clientConn, c.callOptions...)
	}
	return serverConn, serverErr
}

func (c *connectServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	_, clientErr := c.client.Close(ctx, conn, c.callOptions...)
	_, serverErr := next.Server(ctx).Close(ctx, conn)
	if clientErr != nil && serverErr != nil {
		return nil, errors.Wrapf(serverErr, "errors during client close: %v", clientErr)
	}
	if clientErr != nil {
		return nil, errors.Wrap(clientErr, "errors during client close")
	}
	return &empty.Empty{}, serverErr
}
