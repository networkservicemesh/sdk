// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package monitor provides a NetworkServiceServer chain element to provide a monitor server that reflects
// the connections actually in the NetworkServiceServer
package monitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type monitorServer struct {
	networkservice.MonitorConnectionServer
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//             - monitorServerPtr - *networkservice.MonitorConnectionServer.  Since networkservice.MonitorConnectionServer is an interface
//                        (and thus a pointer) *networkservice.MonitorConnectionServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.MonitorConnectionServer
//                        This is done so that we can preserve the return of networkservice.NetworkServer and use
//                        NewServer(...) as any other chain element constructor, but also get back a
//                        networkservice.MonitorConnectionServer that can be used either standalone or in a
//                        networkservice.MonitorConnectionServer chain
//             chainCtx - context for lifecycle management
func NewServer(ctx context.Context, monitorServerPtr *networkservice.MonitorConnectionServer) networkservice.NetworkServiceServer {
	*monitorServerPtr = newMonitorConnectionServer(ctx)
	return &monitorServer{
		MonitorConnectionServer: *monitorServerPtr,
	}
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = WithEventConsumer(ctx, m.MonitorConnectionServer.(EventConsumer))
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		_ = m.MonitorConnectionServer.(EventConsumer).Send(&networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_UPDATE,
			Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := next.Server(ctx).Close(ctx, conn)
	_ = m.MonitorConnectionServer.(EventConsumer).Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
	})
	return rv, err
}
