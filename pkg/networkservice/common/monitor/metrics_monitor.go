// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"github.com/networkservicemesh/api/pkg/api/connection"
)

// MetricsMonitor - extension to monitor service to pass metrics into MonitorServer recipients.
type MetricsMonitor interface {
	// HandleMetrics - send statistics into clients.
	HandleMetrics(statistics map[string]*connection.Metrics)
}

// MetricsServer - a server to send metrics to
type MetricsServer interface {
	// SendMetrics - send metrics to server, bypass selector and match using connections from list of all connections.
	SendMetrics(event *connection.ConnectionEvent, connections map[string]*connection.Connection) error
}
