// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package monitor

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type monitorConnectionServer struct {
	chainCtx    context.Context
	connections map[string]*networkservice.Connection
	filters     map[string]*monitorFilter
	executor    serialize.Executor
}

func newMonitorConnectionServer(chainCtx context.Context) networkservice.MonitorConnectionServer {
	return &monitorConnectionServer{
		chainCtx:    chainCtx,
		connections: make(map[string]*networkservice.Connection),
		filters:     make(map[string]*monitorFilter),
	}
}

func (m *monitorConnectionServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
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
	case <-m.chainCtx.Done():
	}

	return nil
}

var _ networkservice.MonitorConnectionServer = &monitorConnectionServer{}

func (m *monitorConnectionServer) Send(event *networkservice.ConnectionEvent) (_ error) {
	m.executor.AsyncExec(func() {
		if event.Type == networkservice.ConnectionEventType_UPDATE {
			for _, conn := range event.GetConnections() {
				m.connections[conn.GetId()] = conn.Clone()
			}
		}
		if event.Type == networkservice.ConnectionEventType_DELETE {
			for _, conn := range event.GetConnections() {
				delete(m.connections, conn.GetId())
			}
		}
		if event.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER {
			// sending event with INIITIAL_STATE_TRANSFER not permitted
			return
		}
		for id, filter := range m.filters {
			id, filter := id, filter
			e := event.Clone()
			filter.executor.AsyncExec(func() {
				var err error
				select {
				case <-filter.Context().Done():
					m.executor.AsyncExec(func() {
						delete(m.filters, id)
					})
				default:
					err = filter.Send(e)
				}
				if err != nil {
					m.executor.AsyncExec(func() {
						delete(m.filters, id)
					})
				}
			})
		}
	})
	return nil
}

// EventConsumer - interface for monitor events sending
type EventConsumer interface {
	Send(event *networkservice.ConnectionEvent) (err error)
}

var _ EventConsumer = &monitorConnectionServer{}
