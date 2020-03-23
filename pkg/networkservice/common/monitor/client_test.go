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

package monitor_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

const (
	waitForTimeout = 5 * time.Second
	tickTimeout    = 10 * time.Millisecond
)

type mockConnectionEventHandler struct {
	EventCh     chan *networkservice.ConnectionEvent
	BreakdownCh chan struct{}
}

func (m mockConnectionEventHandler) HandleEvent(event *networkservice.ConnectionEvent) {
	m.EventCh <- event
}

func (m mockConnectionEventHandler) HandleMonitorBreakdown() {
	m.BreakdownCh <- struct{}{}
}

func TestNewClient_PassConnectionEventsToHandler(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 1)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	eventHandler := &mockConnectionEventHandler{
		EventCh:     make(chan *networkservice.ConnectionEvent),
		BreakdownCh: make(chan struct{}),
	}
	_ = monitor.NewClient(ctx, eventchannel.NewMonitorConnectionClient(eventCh), eventHandler)

	conn := &networkservice.Connection{
		Id: "conn-1", NetworkService: "ns-1",
	}

	events := []*networkservice.ConnectionEvent{
		{
			Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
			Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
		},
		{
			Type:        networkservice.ConnectionEventType_UPDATE,
			Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
		},
		{
			Type:        networkservice.ConnectionEventType_DELETE,
			Connections: map[string]*networkservice.Connection{conn.GetId(): conn},
		},
	}
	for _, e := range events {
		eventCh <- e
	}

	expectedEventIndex := 0
	cond := func() bool {
		select {
		case r := <-eventHandler.EventCh:
			if reflect.DeepEqual(events[expectedEventIndex], r) {
				expectedEventIndex++
				return true
			}
			return false
		default:
			return false
		}
	}
	require.Eventually(t, cond, waitForTimeout, tickTimeout)
	require.Eventually(t, cond, waitForTimeout, tickTimeout)
	require.Eventually(t, cond, waitForTimeout, tickTimeout)
	assert.Equal(t, len(events), expectedEventIndex)
}

func TestNewClient_PassEventAfterConnectionBreakdown(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 1)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	eventHandler := &mockConnectionEventHandler{
		EventCh:     make(chan *networkservice.ConnectionEvent),
		BreakdownCh: make(chan struct{}),
	}
	_ = monitor.NewClient(ctx, eventchannel.NewMonitorConnectionClient(eventCh), eventHandler)

	close(eventCh)

	breakdownCond := func() bool {
		select {
		case <-eventHandler.BreakdownCh:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, breakdownCond, waitForTimeout, tickTimeout)
}
