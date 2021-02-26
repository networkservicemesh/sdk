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

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type monitorServer struct {
	connections map[string]*networkservice.Connection
	monitors    []networkservice.MonitorConnection_MonitorConnectionsServer
	executor    serialize.Executor
	ctx         context.Context
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
		monitors:    nil, // Intentionally nil
	}
	*monitorServerPtr = rv
	return rv
}

func (m *monitorServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	m.executor.AsyncExec(func() {
		monitor := newMonitorFilter(selector, srv)
		m.monitors = append(m.monitors, monitor)
		connections := make(map[string]*networkservice.Connection)
		for _, ps := range selector.PathSegments {
			if conn, ok := m.connections[ps.GetId()]; ok {
				connections[ps.GetId()] = conn
			}
		}
		// Send initial transfer of all data available
		_ = monitor.Send(&networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: connections,
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
	eventConn := conn.Clone()
	if err == nil {
		m.executor.AsyncExec(func() {
			m.connections[eventConn.GetId()] = eventConn

			// Send update event
			event := &networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: map[string]*networkservice.Connection{eventConn.GetId(): eventConn},
			}
			if sendErr := m.send(ctx, event); sendErr != nil {
				log.FromContext(ctx).Errorf("Error during sending event: %v", sendErr)
			}
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	_, closeErr := next.Server(ctx).Close(ctx, conn)

	// Remove connection object we have and send DELETE
	eventConn := conn.Clone()
	m.executor.AsyncExec(func() {
		delete(m.connections, eventConn.GetId())

		event := &networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_DELETE,
			Connections: map[string]*networkservice.Connection{eventConn.GetId(): eventConn},
		}
		if err := m.send(ctx, event); err != nil {
			log.FromContext(ctx).Errorf("Error during sending event: %v", err)
		}
	})
	return &empty.Empty{}, closeErr
}

// send - perform a send to clients.
func (m *monitorServer) send(ctx context.Context, event *networkservice.ConnectionEvent) (err error) {
	newMonitors := []networkservice.MonitorConnection_MonitorConnectionsServer{}
	for _, filter := range m.monitors {
		select {
		case <-filter.Context().Done():
		default:
			if err = filter.Send(event.Clone()); err != nil {
				log.FromContext(ctx).Errorf("Error sending event: %+v: %+v", event, err)
			}
			newMonitors = append(newMonitors, filter)
		}
	}

	m.monitors = newMonitors
	return err
}
