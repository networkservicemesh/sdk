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
	"github.com/gogo/protobuf/proto"
	"runtime"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type monitorServer struct {
	connections map[string]*networkservice.Connection
	metrics     map[string]*networkservice.Metrics
	monitors    []*monitorFilter
	executor    serialize.Executor
	finalized   chan struct{}

	networkservice.MonitorConnection_MonitorConnectionsServer
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//             - monitorServerPtr - *networkservice.MonitorConnectionServer.  Since networkservice.MonitorConnectionServer is an interface
//                        (and thus a pointer) *networkservice.MonitorConnectionServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.MonitorConnectionServer
//                        This is done so that we can preserve the return of networkservice.NetworkServer and use
//                        NewServer(...) as any other chain element constructor, but also get back a
//                        networkservice.MonitorConnectionServer that can be used either standalone or in a
//                        networkservice.MonitorConnectionServer chain
func NewServer(monitorServerPtr *networkservice.MonitorConnectionServer) networkservice.NetworkServiceServer {
	rv := &monitorServer{
		connections: make(map[string]*networkservice.Connection),
		metrics:     make(map[string]*networkservice.Metrics),
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

func (m *monitorServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	m.executor.AsyncExec(func() {
		monitor := newMonitorFilter(selector, srv)
		m.monitors = append(m.monitors, monitor)
		_ = monitor.Send(&networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: m.connections,
			Metrics:     m.metrics,
		}, m.connections)
	})
	select {
	case <-srv.Context().Done():
	case <-m.finalized:
	}
	return nil
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Pass metrics monitor, so it could be used later in chain.
	ctx = WithServer(ctx, m)

	connectionClone := proto.Clone(request.GetConnection())
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		m.executor.AsyncExec(func() {
			m.connections[conn.GetId()] = conn

			// Send update only if connection is updated.
			if !proto.Equal(connectionClone, conn) {
				event := &networkservice.ConnectionEvent{
					Type:        networkservice.ConnectionEventType_UPDATE,
					Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
				}
				if err = m.send(ctx, event); err != nil {
					logrus.Errorf("Error during sending event: %v", err)
				}
			}
		})
	}
	return conn, err
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Pass metrics monitor, so it could be used later in chain.
	ctx = WithServer(ctx, m)

	m.executor.AsyncExec(func() {
		delete(m.connections, conn.GetId())
		event := &networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_DELETE,
			Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
		}
		if err := m.send(ctx, event); err != nil {
			logrus.Errorf("Error during sending event: %v", err)
		}
	})
	return next.Server(ctx).Close(ctx, conn)
}

func (m *monitorServer) Send(event *networkservice.ConnectionEvent) error {
	m.executor.AsyncExec(func() {
		// we need to update current metrics and connection objects
		if event.GetType() == networkservice.ConnectionEventType_UPDATE {
			for _, conn := range event.GetConnections() {
				m.connections[conn.GetId()] = conn
			}
			for conID, metric := range event.GetMetrics() {
				m.metrics[conID] = metric
			}
		}

		if err := m.send(context.Background(), event); err != nil {
			logrus.Errorf("Error sending event %v", err)
		}
	})
	return nil
}

func (m *monitorServer) send(ctx context.Context, event *networkservice.ConnectionEvent) (err error) {
	m.executor.AsyncExec(func() {
		newMonitors := []*monitorFilter{}
		for _, filter := range m.monitors {
			select {
			case <-filter.srv.Context().Done():
			default:
				if err = filter.Send(event, m.connections); err != nil {
					trace.Log(ctx).Errorf("Error sending event: %+v: %+v", event, err)
				}
				newMonitors = append(newMonitors, filter)
			}
		}
		m.monitors = newMonitors
	})
	return err
}
