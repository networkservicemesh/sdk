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
	"github.com/google/uuid"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type monitorServer struct {
	ctx         context.Context
	connections map[string]*networkservice.Connection
	filters     map[string]*monitorFilter
	executor    serialize.Executor
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//             - monitorServerPtr - *networkservice.MonitorConnectionServer.  Since networkservice.MonitorConnectionServer is an interface
//                        (and thus a pointer) *networkservice.MonitorConnectionServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.MonitorConnectionServer
//                        This is done so that we can preserve the return of networkservice.NetworkServer and use
//                        NewServer(...) as any other chain element constructor, but also get back a
//                        networkservice.MonitorConnectionServer that can be used either standalone or in a
//                        networkservice.MonitorConnectionServer chain
//             ctx - context for lifecycle management
func NewServer(ctx context.Context, monitorServerPtr *networkservice.MonitorConnectionServer) networkservice.NetworkServiceServer {
	rv := &monitorServer{
		ctx:         ctx,
		connections: make(map[string]*networkservice.Connection),
		filters:     make(map[string]*monitorFilter),
	}

	*monitorServerPtr = rv

	return rv
}

func (m *monitorServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	m.executor.AsyncExec(func() {
		filter := newMonitorFilter(selector, srv)
		m.filters[uuid.New().String()] = filter

		connections := networkservice.FilterMapOnManagerScopeSelector(m.connections, selector)

		// Send initial transfer of all data available
		filter.executor.AsyncExec(func() {
			_ = filter.Send(&networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
				Connections: connections,
			})
		})
	})

	select {
	case <-srv.Context().Done():
	case <-m.ctx.Done():
	}

	return nil
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		eventConn := conn.Clone()
		m.executor.AsyncExec(func() {
			m.connections[eventConn.GetId()] = eventConn
			// Send UPDATE
			m.send(ctx, &networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: map[string]*networkservice.Connection{eventConn.GetId(): eventConn},
			})
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	rv, err := next.Server(ctx).Close(ctx, conn)

	// Remove connection object we have and send DELETE
	eventConn := conn.Clone()
	m.executor.AsyncExec(func() {
		delete(m.connections, eventConn.GetId())
		m.send(ctx, &networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_DELETE,
			Connections: map[string]*networkservice.Connection{eventConn.GetId(): eventConn},
		})
	})

	return rv, err
}

func (m *monitorServer) send(ctx context.Context, event *networkservice.ConnectionEvent) {
	logger := log.FromContext(ctx).WithField("monitorServer", "send")
	for id, filter := range m.filters {
		id, filter := id, filter
		e := event.Clone()
		filter.executor.AsyncExec(func() {
			var err error
			select {
			case <-filter.Context().Done():
				err = filter.Context().Err()
			default:
				err = filter.Send(e)
			}
			if err == nil {
				return
			}

			logger.Errorf("error sending event: %+v %s", e, err.Error())
			m.executor.AsyncExec(func() {
				delete(m.filters, id)
			})
		})
	}
}
