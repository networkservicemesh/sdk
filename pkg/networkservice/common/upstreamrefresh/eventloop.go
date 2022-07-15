// Copyright (c) 2022 Cisco and/or its affiliates.
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

package upstreamrefresh

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type eventLoop struct {
	eventLoopCtx  context.Context
	conn          *networkservice.Connection
	eventFactory  begin.EventFactory
	client        networkservice.MonitorConnection_MonitorConnectionsClient
	localNotifier *notifier
	logger        log.Logger
}

func newEventLoop(ctx context.Context, cc grpc.ClientConnInterface, conn *networkservice.Connection, ln *notifier) (context.CancelFunc, error) {
	conn = conn.Clone()

	ev := begin.FromContext(ctx)
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

	logger := log.FromContext(ctx).WithField("upstreamrefresh", "eventLoop")
	cev := &eventLoop{
		eventLoopCtx:  eventLoopCtx,
		conn:          conn,
		eventFactory:  ev,
		client:        newClientFilter(client, conn, logger),
		localNotifier: ln,
		logger:        logger,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) eventLoop() {
	upstreamCh := cev.monitorUpstream()
	var localCh <-chan struct{}

	if cev.localNotifier.get(cev.conn.GetId()) != nil {
		localCh = cev.localNotifier.get(cev.conn.GetId())
	}

	select {
	case _, ok := <-upstreamCh:
		if !ok {
			// Connection closed
			return
		}
		cev.logger.Debug("refresh requested from upstream")

		<-cev.eventFactory.Request()
		cev.localNotifier.Notify(cev.eventLoopCtx, cev.conn.GetId())

	case _, ok := <-localCh:
		if !ok {
			// Unsubscribed
			return
		}
		cev.logger.Debug("refresh requested from other connection")
		<-cev.eventFactory.Request()
	case <-cev.eventLoopCtx.Done():
		return
	}
}

func (cev *eventLoop) monitorUpstream() <-chan struct{} {
	res := make(chan struct{}, 1)

	go func() {
		defer close(res)

		for {
			eventIn, err := cev.client.Recv()
			if cev.eventLoopCtx.Err() != nil || err != nil {
				return
			}

			// Handle event
			if eventIn.GetConnections()[cev.conn.GetId()].GetState() == networkservice.State_REFRESH_REQUESTED {
				res <- struct{}{}
				return
			}
		}
	}()

	return res
}
