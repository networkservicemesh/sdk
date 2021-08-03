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

// Package checkconn - provides networkservice chain element that checks the connection on the next hop
package checkconn

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type checkconnClient struct {
	monitorClient networkservice.MonitorConnectionClient
}

// NewClient - returns a new checkconn networkservicemesh.NetworkServiceClient
func NewClient(monitorClient networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &checkconnClient{
		monitorClient: monitorClient,
	}
}

func (c *checkconnClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if request.GetConnection().GetNextPathSegment() != nil {
		return next.Client(ctx).Request(ctx, request, opts...)
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err == nil {
		return conn, err
	}

	// TODO: replace context.Background() (https://github.com/networkservicemesh/sdk/issues/1026)
	closeCtx := context.Background()
	stream, e := c.monitorClient.MonitorConnections(closeCtx, &networkservice.MonitorScopeSelector{
		PathSegments: request.GetConnection().GetPath().GetPathSegments(),
	})
	if e != nil {
		return nil, errors.Errorf("%v; %v", err, e)
	}
	event, e := stream.Recv()
	if e != nil {
		return nil, errors.Errorf("%v; %v", err, e)
	}

	for _, conn := range event.Connections {
		if conn.GetPrevPathSegment().GetId() == request.GetConnection().GetId() {
			_, _ = next.Client(ctx).Close(closeCtx, conn)
			break
		}
	}
	return conn, err
}

func (c *checkconnClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
