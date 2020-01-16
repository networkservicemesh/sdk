// Copyright (c) 2020 Cisco Systems, Inc.
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
	"runtime"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservicemesh/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type monitorServer struct {
	connections map[string]*connection.Connection
	monitors    []connection.MonitorConnection_MonitorConnectionsServer
	executor    serialize.Executor
	finalized   chan struct{}
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//             - monitorServerPtr - *connection.MonitorConnectionServer.  Since connection.MonitorConnectionServer is an interface
//                        (and thus a pointer) *connection.MonitorConnectionServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements connection.MonitorConnectionServer
//                        This is done so that we can preserve the return of networkservice.NetworkServer and use
//                        NewServer(...) as any other chain element constructor, but also get back a
//                        connection.MonitorConnectionServer that can be used either standalone or in a
//                        connection.MonitorConnectionServer chain
func NewServer(monitorServerPtr *connection.MonitorConnectionServer) networkservice.NetworkServiceServer {
	rv := &monitorServer{
		connections: make(map[string]*connection.Connection),
		monitors:    nil, // Intentionally nil
		executor:    serialize.NewExecutor(),
		finalized:   make(chan struct{}),
	}
	runtime.SetFinalizer(rv, func(server *monitorServer) {
		close(server.finalized)
	})
	*monitorServerPtr = rv
	return rv
}

func (m *monitorServer) MonitorConnections(selector *connection.MonitorScopeSelector, srv connection.MonitorConnection_MonitorConnectionsServer) error {
	m.executor.AsyncExec(func() {
		monitor := newMonitorFilter(selector, srv)
		m.monitors = append(m.monitors, monitor)
		_ = monitor.Send(&connection.ConnectionEvent{
			Type:        connection.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: m.connections,
		})
	})
	select {
	case <-srv.Context().Done():
	case <-m.finalized:
	}
	return nil
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		m.executor.AsyncExec(func() {
			m.connections[conn.GetId()] = conn
			event := &connection.ConnectionEvent{
				Type:        connection.ConnectionEventType_UPDATE,
				Connections: map[string]*connection.Connection{conn.GetId(): conn},
			}
			m.send(ctx, event)
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	m.executor.AsyncExec(func() {
		delete(m.connections, conn.GetId())
		event := &connection.ConnectionEvent{
			Type:        connection.ConnectionEventType_DELETE,
			Connections: map[string]*connection.Connection{conn.GetId(): conn},
		}
		m.send(ctx, event)
	})
	return next.Server(ctx).Close(ctx, conn)
}

func (m *monitorServer) send(ctx context.Context, event *connection.ConnectionEvent) {
	newMonitors := make([]connection.MonitorConnection_MonitorConnectionsServer, len(m.monitors))
	for _, srv := range m.monitors {
		select {
		case <-srv.Context().Done():
		default:
			if err := srv.Send(event); err != nil {
				trace.Log(ctx).Errorf("Error sending event: %+v: %+v", event, err)
			}
			newMonitors = append(newMonitors, srv)
		}
	}
	m.monitors = newMonitors
}
