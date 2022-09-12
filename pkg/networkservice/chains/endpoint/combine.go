// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2022 Cisco Systems, Inc.
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

package endpoint

import "github.com/networkservicemesh/api/pkg/api/networkservice"

// Combine returns a new combined endpoint:
// * networkservice.NetworkServiceServer created by combineFun(eps)
// * networkservice.MonitorConnectionServer part is managed in the following way:
//   - networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER is merged to single event from all endpoints
//   - rest events just go with no changes from all endpoints
func Combine(combineFun func(servers []networkservice.NetworkServiceServer) networkservice.NetworkServiceServer, eps ...Endpoint) Endpoint {
	var servers []networkservice.NetworkServiceServer
	monitorServers := make(map[networkservice.MonitorConnectionServer]int)
	for _, ep := range eps {
		servers = append(servers, ep)
		if _, ok := monitorServers[ep]; !ok {
			monitorServers[ep] = len(monitorServers)
		}
	}

	return &endpoint{
		NetworkServiceServer: combineFun(servers),
		MonitorConnectionServer: &combineMonitorServer{
			monitorServers: monitorServers,
		},
	}
}
