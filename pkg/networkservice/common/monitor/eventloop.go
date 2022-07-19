// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package monitor

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type eventLoop struct {
	eventLoopCtx  context.Context
	conn          *networkservice.Connection
	eventConsumer EventConsumer
	client        networkservice.MonitorConnection_MonitorConnectionsClient
}

func newEventLoop(ctx context.Context, ec EventConsumer, cc grpc.ClientConnInterface, conn *networkservice.Connection) (context.CancelFunc, error) {
	conn = conn.Clone()
	// Is another chain element asking for events?  If not, no need to monitor
	if ec == nil {
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
		eventLoopCtx:  eventLoopCtx,
		conn:          conn,
		eventConsumer: ec,
		client:        newClientFilter(client, conn),
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) eventLoop() {
	// So we have a client, and can receive events
	for {
		eventIn, err := cev.client.Recv()
		if cev.eventLoopCtx.Err() != nil {
			return
		}
		if err != nil {
			// If we get an error, we've lost our connection... Send Down update
			connOut := cev.conn.Clone()
			connOut.State = networkservice.State_DOWN
			eventOut := &networkservice.ConnectionEvent{
				Type: networkservice.ConnectionEventType_UPDATE,
				Connections: map[string]*networkservice.Connection{
					cev.conn.GetId(): connOut,
				},
			}
			_ = cev.eventConsumer.Send(eventOut)
			return
		}
		_ = cev.eventConsumer.Send(eventIn)
	}
}
