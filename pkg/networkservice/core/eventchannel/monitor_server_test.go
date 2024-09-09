// Copyright (c) 2020 Cisco and/or its affiliates.
//
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

package eventchannel_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

const (
	numSenders = 10
	numEvents  = 10
)

func TestMonitorConnectionMonitorConnectionsServer_Send(t *testing.T) {
	testSend := func() {
		eventsCh := make(chan *networkservice.ConnectionEvent, numEvents)
		senderCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		sender := eventchannel.NewMonitorConnectionMonitorConnectionsServer(senderCtx, eventsCh)
		go func() {
			for i := 0; i < numEvents-1 && senderCtx.Err() == nil; i++ {
				err := sender.Send(&networkservice.ConnectionEvent{
					Type: networkservice.ConnectionEventType_UPDATE,
					Connections: map[string]*networkservice.Connection{
						"1": {
							Id: "1",
						},
					},
				})
				if err != nil {
					close(eventsCh)
					return
				}
				<-time.After(time.Millisecond)
			}
		}()
	}
	for now := time.Now(); time.Since(now) < time.Second; {
		testSend()
	}
}

func TestMonitorConnectionServer_MonitorConnections(t *testing.T) {
	senders := make([]networkservice.MonitorConnection_MonitorConnectionsServer, numSenders)
	senderEventChs := make([]chan *networkservice.ConnectionEvent, numSenders)
	senderCancelFunc := make([]context.CancelFunc, numSenders)

	eventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	selector := &networkservice.MonitorScopeSelector{} //nolint:nolintlint // TODO

	eventMonitorStartCh := make(chan int, numEvents)

	server := eventchannel.NewMonitorServer(eventCh, eventchannel.WithConnectChannel(eventMonitorStartCh))

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
	// Give all the go functions calling server.MonitorConnections(selector,sender) a chance to run
	requireConnectionCount(t, numSenders, eventMonitorStartCh)
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

func requireConnectionCount(t *testing.T, expected int, ch <-chan int) {
	deadlineCh := time.After(time.Second)
	var connectionCount int
	for connectionCount != expected {
		select {
		case connectionCount = <-ch:
		case <-deadlineCh:
			require.Failf(t, "Deadline has been reached", "Actual: %v, Expected: %v.", connectionCount, numSenders)
		}
	}
}
