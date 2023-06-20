// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package reselectcleanup provides a server chain element
// that will call Close before request
// if an already existing connection has reselect flag
package reselectcleanup

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type reselectcleanupServer struct {
	connectionProvider monitor.ConnectionProvider
}

// NewServer - create a new reselectcleanup server
func NewServer(connectionProvider monitor.ConnectionProvider) networkservice.NetworkServiceServer {
	return &reselectcleanupServer{
		connectionProvider: connectionProvider,
	}
}

func (c *reselectcleanupServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	if request.GetConnection().GetState() != networkservice.State_RESELECT_REQUESTED {
		return next.Server(ctx).Request(ctx, request)
	}

	conns, err := c.connectionProvider.Find(&networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Id:   request.GetConnection().GetCurrentPathSegment().GetId(),
				Name: request.GetConnection().GetCurrentPathSegment().GetName(),
			},
		},
	})
	if err != nil {
		log.FromContext(ctx).Errorf("Can't check if we have an old connection to close on reselect: %v", err)
		conn, err2 := next.Server(ctx).Request(ctx, request)
		if err2 == nil {
			conn.State = networkservice.State_UP
		}
		return conn, err2
	}
	oldConnection, ok := conns[request.GetConnection().GetId()]
	if !ok || oldConnection == nil {
		// most likely the connection has already been closed
		conn, err2 := next.Server(ctx).Request(ctx, request)
		if err2 == nil {
			conn.State = networkservice.State_UP
		}
		return conn, err2
	}
	log.FromContext(ctx).Info("Closing connection due to RESELECT_REQUESTED state")
	_, err = next.Server(ctx).Close(ctx, oldConnection)
	if err != nil {
		log.FromContext(ctx).Errorf("Can't close old connection: %v", err)
	}
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		conn.State = networkservice.State_UP
	}
	return conn, err
}

func (c *reselectcleanupServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn)
}
