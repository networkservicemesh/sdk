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

package monitor

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type clientEventLoop struct {
	eventLoopCtx   context.Context
	conn           *networkservice.Connection
	eventConsumers []EventConsumer
	client         networkservice.MonitorConnection_MonitorConnectionsClient
}

func newClientEventLoop(ctx context.Context, eventConsumers []EventConsumer, cc grpc.ClientConnInterface, conn *networkservice.Connection) (context.CancelFunc, error) {
	conn = conn.Clone()
	// Is another chain element asking for events?  If not, no need to monitor
	if eventConsumers == nil {
		return func() {}, nil
	}

	// Wrap the event consumers in a filter to translate the event
	var translatedEventConsumers []EventConsumer
	for _, ec := range eventConsumers {
		translatedEventConsumers = append(translatedEventConsumers, newDownstreamEventTranslator(conn, ec))
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

	client, err := networkservice.NewMonitorConnectionClient(cc).MonitorConnections(eventLoopCtx, selector, grpc.WaitForReady(false))
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

	cev := &clientEventLoop{
		eventLoopCtx:   eventLoopCtx,
		conn:           conn,
		eventConsumers: translatedEventConsumers,
		client:         client,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *clientEventLoop) eventLoop() {
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
			for _, eventConsumer := range cev.eventConsumers {
				_ = eventConsumer.Send(eventOut)
			}
			return
		}
		for _, eventConsumer := range cev.eventConsumers {
			_ = eventConsumer.Send(eventIn)
		}
	}
}
