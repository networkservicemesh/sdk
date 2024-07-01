// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"google.golang.org/grpc"
)

type eventLoop struct {
	eventLoopCtx  context.Context
	conn          *networkservice.Connection
	eventConsumer EventConsumer
	cancel        func()
	cc            grpc.ClientConnInterface
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
	cev := &eventLoop{
		eventLoopCtx:  eventLoopCtx,
		conn:          conn,
		eventConsumer: ec,
		cc:            cc,
		cancel:        eventLoopCancel,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) eventLoop() {
	selector := &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			{
				Id:   cev.conn.GetCurrentPathSegment().GetId(),
				Name: cev.conn.GetCurrentPathSegment().GetName(),
			},
		},
	}

	client, err := networkservice.NewMonitorConnectionClient(cev.cc).MonitorConnections(cev.eventLoopCtx, selector)
	if err != nil {
		log.FromContext(cev.eventLoopCtx).Infof("failed to get a MonitorConnections client: %s", err.Error())
		cev.cancel()
		return
	}

	if client == nil {
		log.FromContext(cev.eventLoopCtx).Infof("failed to get a MonitorConnections client: client is nil")
		cev.cancel()
		return
	}

	filter := newClientFilter(client, cev.conn)

	// So we have a client, and can receive events
	for {
		eventIn, err := filter.Recv()
		if cev.eventLoopCtx.Err() != nil {
			return
		}

		connOut := cev.conn.Clone()
		if err != nil && connOut != nil {
			// If we get an error, we've lost our connection... Send Down update
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
