// Copyright (c) 2020 Cisco and/or its affiliates.
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

package adapters

import (
	"context"
	"runtime"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

type monitorServerToClient struct {
	server networkservice.MonitorConnectionServer
}

// NewMonitorServerToClient - returns a MonitorConnectionClient that is a wrapper around the MonitorConnectionServer
//                            events sent to the MonitorConnectionServer are received byt the MonitorConnectionClient
func NewMonitorServerToClient(server networkservice.MonitorConnectionServer) networkservice.MonitorConnectionClient {
	return &monitorServerToClient{server: server}
}

func (m *monitorServerToClient) MonitorConnections(ctx context.Context, selector *networkservice.MonitorScopeSelector, _ ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	eventCh := make(chan *networkservice.ConnectionEvent, 1)
	srv := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	go func() {
		_ = m.server.MonitorConnections(selector, srv)
	}()
	for len(eventCh) == 0 {
		runtime.Gosched()
	}
	return eventchannel.NewMonitorConnectionMonitorConnectionsClient(ctx, eventCh), nil
}
