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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
	"github.com/sirupsen/logrus"
	"runtime"
	"time"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type metricsServer struct {
	metrics   map[string]*metricsMonitor
	executor  serialize.Executor
	finalized chan struct{}
}

type metricsMonitor struct {
	connection *networkservice.Connection
	index      uint32
	metrics    *networkservice.Metrics
	interval   time.Duration
	enabled    bool
	server     networkservice.MonitorConnection_MonitorConnectionsServer
	executor   serialize.Executor
	networkservice.MonitorConnection_MonitorConnectionsServer
	lastSend time.Time
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

	// We need to hook for Send() events to process upcoming monitoring events.
	ctx = monitor.WithServer(ctx, metricsMonitor)

	return next.Server(ctx).Request(ctx, request)
}

func (m *metricsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Pass metrics monitor, so it could be used later in chain.
	var metricsMon *metricsMonitor
	m.executor.SyncExec(func() {
		metricsMon = m.metrics[conn.GetId()]
		ctx = WithServer(ctx, metricsMon)
	})
	empty, err := next.Server(ctx).Close(ctx, conn)
	m.executor.AsyncExec(func() {
		delete(m.metrics, conn.GetId())
	})
	return empty, err
}

/**
configureMonitor - cancel previous go routine with metrics sending for this connection and trigger new one if required.
*/
func (m *metricsMonitor) configureMonitor(conn *networkservice.Connection) {
	m.interval = time.Second * 5 // Default interval is 5 seconds
	m.enabled = false
	// If enabled or not specified
	if conn.Context.MetricsContext != nil && conn.Context.MetricsContext.Enabled {
		m.enabled = true
		intvalue := conn.Context.MetricsContext.Interval
		if intvalue > 0 {
			m.interval = time.Duration(intvalue)
		}
	}
	// Build segments
	m.metrics = &networkservice.Metrics{
		MetricsSegment: []*networkservice.MetricsSegment{},
	}
	m.updateMetricsPath(m.metrics, conn)
	m.lastSend = time.Now()
}

func (m *metricsServer) newMetricsMonitor(conn *networkservice.Connection, monitorServer networkservice.MonitorConnection_MonitorConnectionsServer) (result *metricsMonitor) {
	m.executor.SyncExec(func() {
		result = m.metrics[conn.GetId()]
		if result == nil {
			result = &metricsMonitor{
				connection: conn,
				server:     monitorServer,
				executor:   m.executor,
				index:      conn.GetPath().GetIndex(),
			}
			m.metrics[conn.GetId()] = result
		}
		// Re configure monitor
		result.configureMonitor(conn)
	})
	return result
}

func (m *metricsServer) collectMetrics() map[string]*networkservice.Metrics {
	result := map[string]*networkservice.Metrics{}
	for _, m := range m.metrics {
		result[m.connection.GetId()] = m.metrics
	}
	return result
}

/**
HandleMetrics - update metrics for current connection and current segment index.
*/
func (m *metricsMonitor) HandleMetrics(metrics map[string]string) {
	m.executor.AsyncExec(func() {
		// Put curMetrics
		// We need to construct curMetrics segments to be same as connection ones.
		m.updateMetricsPath(m.metrics, m.connection)
		// replace curMetrics segment curMetrics
		m.metrics.MetricsSegment[m.index].Metrics = metrics

		// Next triggered event will send update and if enabled
		if m.enabled && time.Now().Sub(m.lastSend) > m.interval {
			m.lastSend = time.Now()
			event := &networkservice.ConnectionEvent{
				Type:    networkservice.ConnectionEventType_UPDATE,
				Metrics: map[string]*networkservice.Metrics{m.connection.GetId(): m.metrics},
			}
			m.server.Send(event)
		}
	})
}

func (m *metricsMonitor) updateMetricsPath(curMetrics *networkservice.Metrics, conn *networkservice.Connection) {
	for len(curMetrics.MetricsSegment) < len(conn.Path.PathSegments) {
		newIdx := len(curMetrics.MetricsSegment)
		curMetrics.MetricsSegment = append(curMetrics.MetricsSegment, &networkservice.MetricsSegment{
			Id:   conn.Path.PathSegments[newIdx].Id,
			Name: conn.Path.PathSegments[newIdx].Name,
		})
	}
}

func (m *metricsMonitor) Send(event *networkservice.ConnectionEvent) error {
	m.executor.AsyncExec(func() {
		// Update connection object if passed
		hasConnection := false
		if conn, ok := event.Connections[m.connection.GetId()]; ok {
			m.connection = conn
			hasConnection = true
		}
		// If event had metrics, we need to update them and send only of interval is passed.
		if metricEvt, ok := event.Metrics[m.connection.GetId()]; ok {
			// We need to be sure we replace current metrics with new ones.
			m.updateMetricsPath(m.metrics, m.connection)
			if len(metricEvt.MetricsSegment) >= len(m.metrics.MetricsSegment) {
				m.metrics.MetricsSegment = append(m.metrics.MetricsSegment[0:m.index], metricEvt.MetricsSegment[m.index:len(metricEvt.MetricsSegment)]...)
			}
			// Delete event from metrics
			delete(event.Metrics, m.connection.GetId())
		}

		// If connection update is coming, we need to add our collected metrics in any case.
		// Or we have interval passed so we need to send update
		if m.enabled && (hasConnection || time.Now().Sub(m.lastSend) > m.interval) {
			event.Metrics[m.connection.GetId()] = m.metrics
			m.lastSend = time.Now()
		}

		// Just pass event to next sender
		if err := m.server.Send(event); err != nil {
			logrus.Errorf("Failed to send event to Monitor.Server %v", err)
		}
	})
	return nil
}
