// Copyright (c) 2021 Doc.ai and/or its affiliates.
//
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

package eventchannel

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type monitorConnectionMonitorConnectionsServer struct {
	ctx     context.Context
	eventCh chan<- *networkservice.ConnectionEvent
}

// NewMonitorConnectionMonitorConnectionsServer - returns a networkservice.MonitorConnection_MonitorConnectionsServer
//
//	eventCh - when an event is passed to the Send() method, it is inserted
//	into eventCh
func NewMonitorConnectionMonitorConnectionsServer(ctx context.Context, eventCh chan<- *networkservice.ConnectionEvent) networkservice.MonitorConnection_MonitorConnectionsServer {
	rv := &monitorConnectionMonitorConnectionsServer{
		ctx:     ctx,
		eventCh: eventCh,
	}
	return rv
}

func (m *monitorConnectionMonitorConnectionsServer) Send(event *networkservice.ConnectionEvent) error {
	select {
	case <-m.ctx.Done():
		return m.ctx.Err()
	case m.eventCh <- event:
		return nil
	}
}

func (m *monitorConnectionMonitorConnectionsServer) Context() context.Context {
	return m.ctx
}

func (m *monitorConnectionMonitorConnectionsServer) SetHeader(metadata.MD) error {
	return nil
}

func (m *monitorConnectionMonitorConnectionsServer) SendHeader(metadata.MD) error {
	return nil
}

func (m *monitorConnectionMonitorConnectionsServer) SetTrailer(metadata.MD) {}

func (m *monitorConnectionMonitorConnectionsServer) SendMsg(msg interface{}) error {
	if event, ok := msg.(*networkservice.ConnectionEvent); ok {
		return m.Send(event)
	}
	return errors.Errorf("Not type networkservice.ConnectionEvent -  msg (%+v)", msg)
}

func (m *monitorConnectionMonitorConnectionsServer) RecvMsg(msg interface{}) error {
	return nil
}
