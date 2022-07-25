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

package next

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

// tailMonitorConnectionsServer is a simple implementation of networkservice.MonitorConnectionServer that is called at the end of a chain
// to insure that we never call a method on a nil object

type tailMonitorConnectionsServer struct{}

func (t *tailMonitorConnectionsServer) MonitorConnections(in *networkservice.MonitorScopeSelector, sv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	return nil
}
