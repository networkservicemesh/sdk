// Copyright (c) 2021 Cisco and/or its affiliates.
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

package heal

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
)

type eventLoop struct {
	eventLoopCtx    context.Context
	conn            *networkservice.Connection
	eventFactory    begin.EventFactory
	client          networkservice.MonitorConnection_MonitorConnectionsClient
	livelinessCheck LivelinessCheck
}

func newEventLoop(ctx context.Context, ev begin.EventFactory, cc grpc.ClientConnInterface, conn *networkservice.Connection, livelinessCheck LivelinessCheck) (context.CancelFunc, error) {
	conn = conn.Clone()
	// Is another chain element asking for events?  If not, no need to monitor
	if ev == nil {
		return func() {}, nil
	}

	// Create new eventLoopCtx and store its eventLoopCancel
	eventLoopCtx, eventLoopCancel := context.WithCancel(ctx)

	// Create selector to only ask for events related to our Connection
	selector := &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Id:   conn.GetCurrentPathSegment().GetId(),
				Name: conn.GetCurrentPathSegment().GetName(),
			},
		},
	}

	client, err := networkservice.NewMonitorConnectionClient(cc).MonitorConnections(eventLoopCtx, selector)
	if err != nil {
		eventLoopCancel()
		return nil, errors.WithStack(err)
	}

	// get the initial state transfer and use it to detect whether we have a real connection or not
	_, err = client.Recv()
	if err != nil {
		eventLoopCancel()
		return nil, errors.WithStack(err)
	}

	cev := &eventLoop{
		eventLoopCtx:    eventLoopCtx,
		conn:            conn,
		eventFactory:    ev,
		client:          newClientFilter(client, conn),
		livelinessCheck: livelinessCheck,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) eventLoop() {
	for {
		eventIn, err := cev.client.Recv()
		if cev.eventLoopCtx.Err() != nil {
			return
		}
		if (err != nil || eventIn.GetConnections()[cev.conn.GetId()].GetState() == networkservice.State_DOWN) &&
			!cev.livelinessCheck(cev.conn) {
			_ = cev.eventFactory.Request(begin.WithReselect())
			return
		}
	}
}
