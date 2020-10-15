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

package eventchannel

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

type monitorConnectionMonitorConnectionsClient struct {
	eventCh    <-chan *networkservice.ConnectionEvent
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewMonitorConnectionMonitorConnectionsClient - returns a networkservice.MonitorConnection_MonitorConnectionsClient
//                                                 ctx - context which if Done will cause Recv to return.
//                                                 eventCh - when an event is sent on eventCh, it is returned by the
//                                                 call to Recv on the networkservice.MonitorConnection_MonitorConnectionsClient
func NewMonitorConnectionMonitorConnectionsClient(ctx context.Context, eventCh <-chan *networkservice.ConnectionEvent) networkservice.MonitorConnection_MonitorConnectionsClient {
	ctx, cancelFunc := context.WithCancel(ctx)
	return &monitorConnectionMonitorConnectionsClient{
		eventCh:    eventCh,
		ctx:        ctx,
		cancelFunc: cancelFunc,
	}
}

func (m *monitorConnectionMonitorConnectionsClient) Recv() (*networkservice.ConnectionEvent, error) {
	select {
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	case event, ok := <-m.eventCh:
		if !ok {
			m.cancelFunc()
			return nil, errors.New("No more events, chan closed by sender")
		}
		return event, nil
	}
}

func (m *monitorConnectionMonitorConnectionsClient) Header() (metadata.MD, error) {
	return make(metadata.MD), nil
}

func (m *monitorConnectionMonitorConnectionsClient) Trailer() metadata.MD {
	return make(metadata.MD)
}

func (m *monitorConnectionMonitorConnectionsClient) CloseSend() error {
	return nil
}

func (m *monitorConnectionMonitorConnectionsClient) Context() context.Context {
	return m.ctx
}

func (m *monitorConnectionMonitorConnectionsClient) SendMsg(msg interface{}) error {
	return nil
}

func (m *monitorConnectionMonitorConnectionsClient) RecvMsg(msg interface{}) error {
	if event, ok := msg.(*networkservice.ConnectionEvent); ok {
		e, err := m.Recv()
		if err != nil {
			return err
		}
		event.Type = e.Type
		event.Connections = e.Connections
		return nil
	}
	return errors.Errorf("Not type networkservice.ConnectionEvent -  msg (%+v)", msg)
}
