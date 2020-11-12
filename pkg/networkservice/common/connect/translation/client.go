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

package translation

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type translationClient struct {
	requestOpts []RequestOption
	connOpts    []ConnectionOption
	clientConns connectionMap
}

func (c *translationClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	connID := request.GetConnection().GetId()

	// 1. Translate request
	clientRequest := request.Clone()
	clientConn, _ := c.clientConns.Load(connID)
	for _, opt := range c.requestOpts {
		opt(clientRequest, clientConn)
	}

	// 2. Request client chain
	clientConn, err = next.Client(ctx).Request(ctx, clientRequest, opts...)
	if err != nil {
		return nil, err
	}
	c.clientConns.Store(connID, clientConn)

	// 3. Translate connection
	conn = request.GetConnection()
	for _, opt := range c.connOpts {
		opt(conn, clientConn)
	}

	return conn, nil
}

func (c *translationClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if clientConn, ok := c.clientConns.LoadAndDelete(conn.GetId()); ok {
		conn = clientConn
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
