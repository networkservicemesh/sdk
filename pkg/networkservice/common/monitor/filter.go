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

package monitor

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type monitorFilter struct {
	selector *networkservice.MonitorScopeSelector
	srv      networkservice.MonitorConnection_MonitorConnectionsServer
}

func newMonitorFilter(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) *monitorFilter {
	return &monitorFilter{
		selector: selector,
		srv:      srv,
	}
}

func (m *monitorFilter) Send(event *networkservice.ConnectionEvent, current map[string]*networkservice.Connection) error {
	// Filter connections based on event passed and selector for this filter
	rv := &networkservice.ConnectionEvent{
		Type:        event.Type,
		Connections: networkservice.FilterMapOnManagerScopeSelector(event.GetConnections(), m.selector),
		Metrics:     filteredMetrics(event, networkservice.FilterMapOnManagerScopeSelector(current, m.selector)),
	}
	if rv.Type == networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER || len(rv.GetConnections()) > 0 || len(rv.GetMetrics()) > 0 {
		return m.srv.Send(rv)
	}
	return nil
}

func filteredMetrics(event *networkservice.ConnectionEvent, matchedConnections map[string]*networkservice.Connection) map[string]*networkservice.Metrics {
	filteredMetrics := map[string]*networkservice.Metrics{}

	// Put only relevant metrics
	for k, v := range event.Metrics {
		if _, ok := matchedConnections[k]; ok {
			filteredMetrics[k] = v
		}
	}
	return filteredMetrics
}
