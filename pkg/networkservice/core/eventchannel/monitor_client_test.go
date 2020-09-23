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

func TestNewMonitorConnectionClient_MonitorConnections(t *testing.T) {
	maxReceivers := 10
	receivers := make([]networkservice.MonitorConnection_MonitorConnectionsClient, maxReceivers)
	cancelFuncs := make([]context.CancelFunc, maxReceivers)

	numEvents := 10
	eventCh := make(chan *networkservice.ConnectionEvent, numEvents)
	client := eventchannel.NewMonitorConnectionClient(eventCh)
	for i := 0; i < maxReceivers; i++ {
		var ctx context.Context
		ctx, cancelFuncs[i] = context.WithCancel(context.Background())
		receiver, err := client.MonitorConnections(ctx, nil)
		receivers[i] = receiver
		assert.Nil(t, err)
		assert.NotNil(t, receivers[i])
	}
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
	for i := 0; i < maxReceivers; i++ {
		for j := 0; j < numEvents; j++ {
			eventOut, err := receivers[i].Recv()
			assert.Nil(t, err)
			assert.Equal(t, eventsIn[j], eventOut)
		}
	}

	cancelFuncs[0]()
	_, err := receivers[0].Recv()
	assert.NotNil(t, err)
	close(eventCh)
}
