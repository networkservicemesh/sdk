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
package metrics

import (
	"context"
	"errors"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
	"github.com/sirupsen/logrus"
	"runtime"
)

type metricsServer struct {
	metrics   map[string]*metricsMonitor
	executor  serialize.Executor
	finalized chan struct{}
}

type metricsMonitor struct {
	connection *networkservice.Connection
	index      uint32
	server     networkservice.MonitorConnection_MonitorConnectionsServer
	executor   serialize.Executor
	networkservice.MonitorConnection_MonitorConnectionsServer
	enabled bool
}

// NewServer - creates a NetworkServiceServer chain element that will properly hold and periodically send metrics to passed MonitorServer
func NewServer() networkservice.NetworkServiceServer {
	rv := &metricsServer{
		metrics:   make(map[string]*metricsMonitor),
		executor:  serialize.NewExecutor(),
		finalized: make(chan struct{}),
	}
	runtime.SetFinalizer(rv, func(server *metricsServer) {
		close(server.finalized)
	})
	return rv
}

func (m *metricsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	monitorServer := monitor.Server(ctx)
	if monitorServer == nil {
		return nil, errors.New("Metrics chain require monitor.Server to be passed")
	}

	// Pass metrics monitor, so it could be used later in chain.
	metricsMonitor := m.newMetricsMonitor(request.GetConnection(), monitorServer)
	ctx = WithServer(ctx, metricsMonitor)

	ctx = monitor.WithServer(ctx, metricsMonitor)

	conn, err := next.Server(ctx).Request(ctx, request)
	if err == nil {
		m.executor.SyncExec(func() {
			metricsMonitor.updateConnection(conn)
		})
		m.executor.AsyncExec(func() {
			metricsMonitor.sendCurrentMetricsEvent()
		})
	}
	return conn, err
}

func (m *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Pass metrics monitor, so it could be used later in chain.
	m.executor.SyncExec(func() {
		metricsMon := m.metrics[conn.GetId()]
		if metricsMon != nil {
			ctx = WithServer(ctx, metricsMon)
		}
	})

	empt, err := next.Server(ctx).Close(ctx, conn)

	m.executor.AsyncExec(func() {
		delete(m.metrics, conn.GetId())
	})

	return empt, err
}
func (m *metricsServer) newMetricsMonitor(conn *networkservice.Connection, monitorServer networkservice.MonitorConnection_MonitorConnectionsServer) (result *metricsMonitor) {
	// New metrics lock for writing.
	m.executor.SyncExec(func() {
		result = m.metrics[conn.GetId()]
		if result == nil {
			result = &metricsMonitor{
				connection: proto.Clone(conn).(*networkservice.Connection),
				server:     monitorServer,
				executor:   m.executor,
				index:      conn.GetPath().GetIndex(),
			}
			m.metrics[conn.GetId()] = result
		}
		// Re configure monitor
		result.enabled = conn.Context.MetricsContext != nil && conn.Context.MetricsContext.Enabled
	})

	return result
}

/**
HandleMetrics - update metrics for current connection and current segment index.
*/
func (m *metricsMonitor) HandleMetrics(metrics map[string]string) {
	m.executor.AsyncExec(func() {
		// Put curMetrics
		// We need to construct curMetrics segments to be same as connection ones.
		// replace curMetrics segment curMetrics
		m.connection.Path.PathSegments[m.index].Metrics = metrics

		// Send to previous monitor if enabled
		if m.enabled {
			m.sendCurrentMetricsEvent()
		}
	})
}

func (m *metricsMonitor) sendCurrentMetricsEvent() {
	event := &networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{m.connection.GetId(): m.connection},
	}
	if err := m.server.Send(event); err != nil {
		logrus.Errorf("Failed to send event to Monitor.Server %v", err)
	}
}

func (m *metricsMonitor) Send(event *networkservice.ConnectionEvent) error {
	m.executor.AsyncExec(func() {
		// Update connection object if passed
		if conn, ok := event.Connections[m.connection.GetId()]; ok {
			m.updateConnection(conn)
		}
		// Just pass event to next sender
		if err := m.server.Send(event); err != nil {
			logrus.Errorf("Failed to send event to Monitor.Server %v", err)
		}
	})
	return nil
}

func (m *metricsMonitor) updateConnection(conn *networkservice.Connection) {
	// We need to preserve current index metrics, since only we provide it.
	currentMetrics := m.connection.GetPath().GetPathSegments()[m.index].Metrics
	conn.GetPath().GetPathSegments()[m.index].Metrics = currentMetrics
	m.connection = proto.Clone(conn).(*networkservice.Connection)
}
