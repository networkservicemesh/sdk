// Copyright (c) 2022 Cisco and/or its affiliates.
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

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
)

type monitorConnectionServer struct {
	chainCtx    context.Context
	connections map[string]*networkservice.Connection
	filters     map[string]*monitorFilter
	executor    *serialize.Executor
}

func newMonitorConnectionServer(chainCtx context.Context, executor *serialize.Executor,
	filters map[string]*monitorFilter, connections map[string]*networkservice.Connection) networkservice.MonitorConnectionServer {
	return &monitorConnectionServer{
		chainCtx:    chainCtx,
		connections: connections,
		filters:     filters,
		executor:    executor,
	}
}

func (m *monitorConnectionServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	if err := next.MonitorConnectionServer(srv.Context()).MonitorConnections(selector, srv); err != nil {
		return err
	}
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
