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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

func TestMonitorConnection_MonitorConnectionsClient_Recv(t *testing.T) {
	numEvents := 50
	eventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)
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
	for i := 0; i < 50; i++ {
		eventOut, err := mcc.Recv()
		assert.Nil(t, err)
		assert.Equal(t, eventsIn[i], eventOut)
	}
	close(eventCh)
	_, err := mcc.Recv()
	assert.NotNil(t, err)
	select {
	case <-mcc.Context().Done():
	default:
		assert.Fail(t, "Context should be Done")
	}
}

func TestMonitorConnection_MonitorConnectionsClient_RecvMsg(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)

	var wrongTypeEvent struct{}
	err := mcc.RecvMsg(wrongTypeEvent)
	assert.NotNil(t, err)

	eventIn := &networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			"foo": {
				Id: "foo",
			},
		},
	}
	eventCh <- eventIn
	eventOut := &networkservice.ConnectionEvent{}
	err = mcc.RecvMsg(eventOut)
	assert.Nil(t, err)
	assert.Equal(t, eventIn, eventOut)

	close(eventCh)
	err = mcc.RecvMsg(eventOut)
	assert.NotNil(t, err)
	select {
	case <-mcc.Context().Done():
	default:
		assert.Fail(t, "Context should be Done")
	}
}

func TestMonitorConnection_MonitorConnectionsClient_CloseSend(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)
	assert.Nil(t, mcc.CloseSend())
}

func TestMonitorConnection_MonitorConnectionsClient_Header(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)
	header, err := mcc.Header()
	assert.Nil(t, err)
	assert.NotNil(t, header)
}

func TestMonitorConnection_MonitorConnectionsClient_SendMsg(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)
	assert.Nil(t, mcc.SendMsg(nil))
}

func TestMonitorConnection_MonitorConnectionsClient_Trailer(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent)
	mcc := eventchannel.NewMonitorConnectionMonitorConnectionsClient(context.Background(), eventCh)
	assert.NotNil(t, mcc.Trailer())
}
