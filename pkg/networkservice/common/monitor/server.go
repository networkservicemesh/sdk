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
	"time"

	"github.com/sirupsen/logrus"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

const defaultMetricsInterval = time.Second * 5

type monitorServer struct {
	connections map[string]*connectionInfo
	monitors    []*monitorFilter
	executor    serialize.Executor
	finalized   chan struct{}

	networkservice.MonitorConnection_MonitorConnectionsServer
}

type connectionInfo struct {
	connection      *networkservice.Connection
	metricsEnabled  bool
	metricsInterval time.Duration
	lastUpdateSend  time.Time
	pendingUpdates  int
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
		connections: make(map[string]*connectionInfo),
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

		// Send initial transfer of all data available
		_ = monitor.Send(&networkservice.ConnectionEvent{
			Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: m.collect(),
		})
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

	// take a copy of connection to be able to check for changes.
	connectionClone := proto.Clone(request.GetConnection()).(*networkservice.Connection)

	// If we have metrics already and they are pending, lets' update current connection we pass next.
	var connInfo *connectionInfo
	m.executor.SyncExec(func() {
		connInfo = m.connections[request.GetConnection().GetId()]
		if connInfo != nil {
			m.updateMetricsInfo(connInfo, request.GetConnection())
		}
	})
	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		m.executor.AsyncExec(func() {
			if connInfo == nil {
				connInfo = m.createConnectionInfo(conn)
			}
			// Send update only if connection is updated or has metrics pending updates
			if !m.updateMetricsInfo(connInfo, conn) && connectionClone.Compare(conn) == networkservice.ConnectionsEqual {
				return
			}
			event := &networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_UPDATE,
				Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
			}
			if err = m.send(ctx, event); err != nil {
				logrus.Errorf("Error during sending event: %v", err)
			}
		})
	}
	return conn, err
}

func (m *monitorServer) createConnectionInfo(conn *networkservice.Connection) *connectionInfo {
	connInfo := &connectionInfo{
		connection:      conn,
		metricsInterval: defaultMetricsInterval,
	}
	m.connections[conn.GetId()] = connInfo
	return connInfo
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// updateMetricsInfo - take metrics statistics and update metrics for conn if pending updates.
func (m *monitorServer) updateMetricsInfo(info *connectionInfo, conn *networkservice.Connection) bool {
	info.metricsEnabled = conn.Context.MetricsContext != nil && conn.Context.MetricsContext.Enabled
	info.metricsInterval = time.Second * 5
	// Update metrics Interval
	if info.metricsEnabled {
		interval := conn.Context.MetricsContext.Interval
		if interval > 0 {
			info.metricsInterval = time.Duration(interval)
		}
	}
	if info.pendingUpdates > 0 {
		// Copy all pending metrics info into connection.
		info.pendingUpdates = 0
		info.lastUpdateSend = time.Now()

		// Update segments with matched ids and name
		connSegmLen := len(conn.GetPath().GetPathSegments())
		for ind, segm := range info.connection.GetPath().GetPathSegments() {
			if ind < connSegmLen {
				conn.GetPath().GetPathSegments()[ind].Metrics = segm.Metrics
			}
		}
	}

	return info.metricsEnabled && info.pendingUpdates > 0
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
				info := m.connections[conn.GetId()]
				if info == nil {
					continue
				}
				// If only metrics are updated, we not required to send update for this connection if interval is not passed
				metricsOnly := info.connection.Compare(conn) == networkservice.ConnectionMetricsUpdated

				info.connection = conn
				// Check interval and force update
				if metricsOnly && info.metricsEnabled && time.Now().Sub(info.lastUpdateSend) > info.metricsInterval {
					// We sending info
					info.lastUpdateSend = time.Now()
					info.pendingUpdates = 0
				} else {
					// we not sending metrics, since connection is same or only metrics are updated.
					info.pendingUpdates++
					delete(event.GetConnections(), conn.GetId())
				}
			}
		}
		if len(event.GetConnections()) == 0 {
			return
		}
		if err := m.send(context.Background(), event); err != nil {
			logrus.Errorf("Error sending event %v", err)
		}
	})
	return nil
}

// send - perform a send to clients.
func (m *monitorServer) send(ctx context.Context, event *networkservice.ConnectionEvent) (err error) {
	newMonitors := []*monitorFilter{}
	for _, filter := range m.monitors {
		select {
		case <-filter.srv.Context().Done():
		default:
			if err = filter.Send(event); err != nil {
				trace.Log(ctx).Errorf("Error sending event: %+v: %+v", event, err)
			}
			newMonitors = append(newMonitors, filter)
		}
	}
	m.monitors = newMonitors
	return err
}

func (m *monitorServer) collect() map[string]*networkservice.Connection {
	connections := map[string]*networkservice.Connection{}
	for _, info := range m.connections {
		connections[info.connection.GetId()] = info.connection
	}
	return connections
}
