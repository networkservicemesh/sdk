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

package eventchannel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

func TestMonitorConnectionServer_MonitorConnections(t *testing.T) {
	numSenders := 10
	senders := make([]networkservice.MonitorConnection_MonitorConnectionsServer, numSenders)
	senderEventChs := make([]chan *networkservice.ConnectionEvent, numSenders)
	senderCancelFunc := make([]context.CancelFunc, numSenders)

	numEvents := 10
	eventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	selector := &networkservice.MonitorScopeSelector{} // TODO

	server := eventchannel.NewMonitorServer(eventCh)

	for i := 0; i < numSenders; i++ {
		var senderCtx context.Context
		senderCtx, senderCancelFunc[i] = context.WithCancel(context.Background())
		senderEventChs[i] = make(chan *networkservice.ConnectionEvent, numEvents)
		sender := eventchannel.NewMonitorConnectionMonitorConnectionsServer(senderCtx, senderEventChs[i])
		senders[i] = sender
		go func() {
			err := server.MonitorConnections(selector, sender)
			assert.Nil(t, err)
		}()
	}
	// Give the go functions calling server.MonitorConnections(selector,sender) a chance to run
	<-time.After(time.Millisecond)
	senderCancelFunc[numSenders-1]()
	eventsIn := make([]*networkservice.ConnectionEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		eventsIn[i] = &networkservice.ConnectionEvent{
			Type: networkservice.ConnectionEventType_UPDATE,
			Connections: map[string]*networkservice.Connection{
				fmt.Sprintf("%d", i): {
					Id: fmt.Sprintf("%d", i),
				},
			},
		}
		eventCh <- eventsIn[i]
	}
	senderCancelFunc[numSenders-2]()
	for i := 0; i < numSenders-2; i++ {
		for j := 0; j < numEvents; j++ {
			event, ok := <-senderEventChs[i]
			assert.True(t, ok)
			assert.Equal(t, eventsIn[j], event)
		}
	}
	close(eventCh)
	senderEventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	srv := eventchannel.NewMonitorConnectionMonitorConnectionsServer(context.Background(), senderEventCh)
	for {
		err := server.MonitorConnections(&networkservice.MonitorScopeSelector{}, srv)
		if err != nil {
			break
		}
	}
}
