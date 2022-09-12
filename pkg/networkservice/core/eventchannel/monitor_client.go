// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// Package eventchannel provides implementations based on event channels of:
//
//	networkservice.MonitorConnectionClient
//	networkservice.MonitorConnectionServer
//	networkservice.MonitorConnection_MonitorConnectionsClient
//	networkservice.MonitorConnection_MonitorConnectionsServer
package eventchannel

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/edwarnicke/serialize"
)

type monitorConnectionClient struct {
	once           sync.Once
	eventCh        <-chan *networkservice.ConnectionEvent
	fanoutEventChs []chan *networkservice.ConnectionEvent
	updateExecutor serialize.Executor
}

// NewMonitorConnectionClient - returns networkservice.MonitorConnectionClient
//
//	eventCh - channel that provides events to feed the Recv function
//	          when an event is sent on the eventCh, all networkservice.MonitorConnection_MonitorConnectionsClient
//	          returned from calling MonitorConnections receive the event.
//	Note: Does not perform filtering basedon MonitorScopeSelector
func NewMonitorConnectionClient(eventCh <-chan *networkservice.ConnectionEvent) networkservice.MonitorConnectionClient {
	return &monitorConnectionClient{
		eventCh: eventCh,
	}
}

func (m *monitorConnectionClient) MonitorConnections(ctx context.Context, _ *networkservice.MonitorScopeSelector, _ ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	fanoutEventCh := make(chan *networkservice.ConnectionEvent, 100)
	m.updateExecutor.AsyncExec(func() {
		m.once.Do(m.eventLoop)
		m.fanoutEventChs = append(m.fanoutEventChs, fanoutEventCh)
		go func() {
			<-ctx.Done()
			m.updateExecutor.AsyncExec(func() {
				if len(m.fanoutEventChs) == 0 {
					return
				}
				var newFanoutEventChs []chan *networkservice.ConnectionEvent
				for _, ch := range m.fanoutEventChs {
					if ch == fanoutEventCh {
						close(fanoutEventCh)
						continue
					}
					newFanoutEventChs = append(newFanoutEventChs, ch)
				}
				m.fanoutEventChs = newFanoutEventChs
			})
		}()
	})
	return NewMonitorConnectionMonitorConnectionsClient(ctx, fanoutEventCh), nil
}

func (m *monitorConnectionClient) eventLoop() {
	go func() {
		for event := range m.eventCh {
			e := event
			m.updateExecutor.AsyncExec(func() {
				for _, fanoutEventCh := range m.fanoutEventChs {
					fanoutEventCh <- e
				}
			})
		}
		m.updateExecutor.AsyncExec(func() {
			for _, fanoutEventCh := range m.fanoutEventChs {
				close(fanoutEventCh)
			}
			m.fanoutEventChs = []chan *networkservice.ConnectionEvent{}
		})
	}()
}
