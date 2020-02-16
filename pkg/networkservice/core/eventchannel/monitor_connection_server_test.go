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
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
)

func TestMonitorConnection_MonitorConnectionsServer_Send(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctx, cancelFunc := context.WithCancel(context.Background())
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	eventIn := &networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			"foo": {
				Id: "foo",
			},
		},
	}
	assert.Nil(t, mcs.Send(eventIn))
	eventOut := <-eventCh
	assert.Equal(t, eventIn, eventOut)
	cancelFunc()
}

func TestMonitorConnection_MonitorConnectionsServer_SendMsg(t *testing.T) {
	numEvents := 50
	eventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	ctx := context.Background()
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)

	var wrongTypeEvent struct{}
	assert.NotNil(t, mcs.SendMsg(wrongTypeEvent))
	eventsIn := make([]*networkservice.ConnectionEvent, numEvents)
	for i := 0; i < numEvents; i++ {
		eventsIn[i] = &networkservice.ConnectionEvent{
			Type: networkservice.ConnectionEventType_UPDATE,
			Connections: map[string]*networkservice.Connection{
				"foo": {
					Id: "foo",
				},
			},
		}
		assert.Nil(t, mcs.SendMsg(eventsIn[i]))
		eventOut := <-eventCh
		assert.Equal(t, eventsIn[i], eventOut)
	}
}

func TestMonitorConnection_MonitorConnectionsServer_RecvMsg(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctx := context.Background()
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	eventIn := &networkservice.ConnectionEvent{}
	assert.Nil(t, mcs.RecvMsg(eventIn))
}

func TestMonitorConnection_MonitorConnectionsServer_Context(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctxIn, cancelFunc := context.WithCancel(context.Background())
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctxIn, eventCh)
	ctxOut := mcs.Context()
	cancelFunc()
	select {
	case <-ctxOut.Done():
	default:
		assert.Fail(t, "Mismatched contexts")
	}
}

func TestMonitorConnection_MonitorConnectionsServer_SendHeader(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctx := context.Background()
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	assert.Nil(t, mcs.SendHeader(make(metadata.MD)))
}

func TestMonitorConnection_MonitorConnectionsServer_SetHeader(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctx := context.Background()
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	assert.Nil(t, mcs.SetHeader(make(metadata.MD)))
}

func TestMonitorConnection_MonitorConnectionsServer_SetTrailer(t *testing.T) {
	eventCh := make(chan *networkservice.ConnectionEvent, 100)
	ctx := context.Background()
	mcs := eventchannel.NewMonitorConnectionMonitorConnectionsServer(ctx, eventCh)
	mcs.SetTrailer(make(metadata.MD))
}
