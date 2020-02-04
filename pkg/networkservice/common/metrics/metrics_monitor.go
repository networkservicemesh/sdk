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

package metrics

// MetricsMonitor - extension to monitor service to pass metrics into MonitorServer recipients.
// monitor will send events with configured with connection interval, often updates will wait for
// next interval approaching before will be sended.
type MetricsMonitor interface {
	// HandleMetrics - send update to current segment metrics
	HandleMetrics(metrics map[string]string)
}
