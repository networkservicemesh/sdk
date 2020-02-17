// Copyright (c) 2020 Cisco and/or its affiliates.
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

package adapters

import (
	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type monitorClientToServer struct {
	client networkservice.MonitorConnectionClient
}

// NewMonitorClientToServer - returns a MonitorConnectionServer that is wrapped around the provided MonitorConnectionClient
//                            events that are received by the MonitorConnectionClient are sent to the MonitorConnectionServer
func NewMonitorClientToServer(client networkservice.MonitorConnectionClient) networkservice.MonitorConnectionServer {
	return &monitorClientToServer{
		client: client,
	}
}

func (m monitorClientToServer) MonitorConnections(selector *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	cl, err := m.client.MonitorConnections(srv.Context(), selector)
	if err != nil {
		return err
	}
	for {
		event, err := cl.Recv()
		if err != nil {
			return err
		}
		err = srv.Send(event)
		if err != nil {
			return err
		}
	}
}
