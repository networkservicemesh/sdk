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

package monitor

import (
	"github.com/edwarnicke/serialize"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type monitorFilter struct {
	selector *networkservice.MonitorScopeSelector
	executor serialize.Executor

	networkservice.MonitorConnection_MonitorConnectionsServer
}

func newMonitorFilter(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) *monitorFilter {
	return &monitorFilter{
		selector: selector,
		MonitorConnection_MonitorConnectionsServer: srv,
	}
}

// Send - Filter connections based on event passed and selector for this filter
func (m *monitorFilter) Send(event *networkservice.ConnectionEvent) error {
	rv := &networkservice.ConnectionEvent{
		Type:        event.Type,
		Connections: networkservice.FilterMapOnManagerScopeSelector(event.GetConnections(), m.selector),
	}
	if rv.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER || len(rv.GetConnections()) > 0 {
		return m.MonitorConnection_MonitorConnectionsServer.Send(rv)
	}
	return nil
}
