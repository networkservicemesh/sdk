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

package heal

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	attemptAfter          = time.Millisecond * 200
	livenessCheckInterval = time.Microsecond * 200
)

type eventLoop struct {
	eventLoopCtx    context.Context
	chainCtx        context.Context
	conn            *networkservice.Connection
	eventFactory    begin.EventFactory
	client          networkservice.MonitorConnection_MonitorConnectionsClient
	livelinessCheck LivelinessCheck
}

func newEventLoop(ctx context.Context, cc grpc.ClientConnInterface, conn *networkservice.Connection, livelinessCheck LivelinessCheck) (context.CancelFunc, error) {
	conn = conn.Clone()

	ev := begin.FromContext(ctx)
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
		chainCtx:        ctx,
		conn:            conn,
		eventFactory:    ev,
		client:          newClientFilter(client, conn, log.FromContext(ctx)),
		livelinessCheck: livelinessCheck,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) waitCtrlPlaneEvent() <-chan struct{} {
	res := make(chan struct{}, 1)

	go func() {
		defer close(res)

		for {
			eventIn, err := cev.client.Recv()

			if cev.chainCtx.Err() != nil || cev.eventLoopCtx.Err() != nil {
				res <- struct{}{}
				return
			}

			// Handle error
			if err != nil {
				s, _ := status.FromError(err)
				// This condition means, that the client closed the connection. Stop healing
				if s.Code() == codes.Canceled {
					res <- struct{}{}
				}
				// Otherwise - Start healing
				return
			}

			// Handle event. Start healing
			if eventIn.GetConnections()[cev.conn.GetId()].GetState() == networkservice.State_DOWN {
				return
			}
		}
	}()

	return res
}

func (cev *eventLoop) waitDataPlaneEvent() <-chan struct{} {
	res := make(chan struct{}, 1)

	go func() {
		defer close(res)

		ticker := clock.FromContext(cev.eventLoopCtx).Ticker(livenessCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cev.chainCtx.Done():
				res <- struct{}{}
				return
			case <-cev.eventLoopCtx.Done():
				res <- struct{}{}
				return
			case <-ticker.C():
				if !cev.livelinessCheck(cev.conn) {
					// Datapath broken, start healing
					println("EVENT Datapath broken, start healing")
					return
				}
			}
		}
	}()

	return res
}

func (cev *eventLoop) eventLoop() {
	reselect := false

	ctrlPlaneCh := cev.waitCtrlPlaneEvent()

	var dataPlaneCh <-chan struct{}
	if cev.livelinessCheck != nil {
		dataPlaneCh = cev.waitDataPlaneEvent()
	} else {
		// Don't start unnecessary goroutine when no livenessCheck provided
		dataPlaneCh = make(<-chan struct{})
		// Since we don't know about datapath status - always use reselect
		reselect = true
	}

	select {
	case _, ok := <-ctrlPlaneCh:
		if ok {
			// Connection closed
			return
		}
		// Start healing
	case _, ok := <-dataPlaneCh:
		if ok {
			// Connection closed
			return
		}
		reselect = true
		// Start healing
	case <-cev.chainCtx.Done():
		return
	case <-cev.eventLoopCtx.Done():
		return
	}

	/* Attempts to heal the connection */
	afterTicker := clock.FromContext(cev.eventLoopCtx).Ticker(attemptAfter)
	defer afterTicker.Stop()
	for {
		select {
		case <-cev.chainCtx.Done():
			return
		case <-afterTicker.C():
			var errCh <-chan error
			if !reselect {
				reselect = !cev.livelinessCheck(cev.conn)
			}
			if reselect {
				errCh = cev.eventFactory.Request(begin.WithReselect())
			} else {
				errCh = cev.eventFactory.Request()
			}
			if err := <-errCh; err == nil {
				return
			}
		}
	}
}
