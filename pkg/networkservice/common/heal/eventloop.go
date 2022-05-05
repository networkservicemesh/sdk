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
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type eventLoop struct {
	heal         *healClient
	eventLoopCtx context.Context
	chainCtx     context.Context
	conn         *networkservice.Connection
	eventFactory begin.EventFactory
	client       networkservice.MonitorConnection_MonitorConnectionsClient
	logger       log.Logger
}

func newEventLoop(ctx context.Context, cc grpc.ClientConnInterface, conn *networkservice.Connection, heal *healClient) (context.CancelFunc, error) {
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

	logger := log.FromContext(ctx).WithField("heal", "eventLoop")
	cev := &eventLoop{
		heal:         heal,
		eventLoopCtx: eventLoopCtx,
		chainCtx:     ctx,
		conn:         conn,
		eventFactory: ev,
		client:       newClientFilter(client, conn, logger),
		logger:       logger,
	}

	// Start the eventLoop
	go cev.eventLoop()
	return eventLoopCancel, nil
}

func (cev *eventLoop) monitorCtrlPlane() <-chan struct{} {
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

func (cev *eventLoop) eventLoop() {
	reselect := false

	ctrlPlaneCh := cev.monitorCtrlPlane()

	livenessCheckCtx, livenessCheckCancel := context.WithCancel(context.Background())
	defer livenessCheckCancel()
	if cev.heal.livenessCheck != nil {
		go func() {
			cev.monitorDataPlane()
			livenessCheckCancel()
		}()
	} else {
		// Since we don't know about data path status - always use reselect
		reselect = true
	}

	select {
	case _, ok := <-ctrlPlaneCh:
		cev.logger.Warnf("Control plane is down")
		if ok {
			// Connection closed
			return
		}
		// Start healing
	case <-livenessCheckCtx.Done():
		cev.logger.Warnf("Data plane is down")
		reselect = true
		// Start healing
	case <-cev.chainCtx.Done():
		return
	case <-cev.eventLoopCtx.Done():
		return
	}

	/* Attempts to heal the connection */
	for {
		select {
		case <-cev.chainCtx.Done():
			return
		default:
			var options []begin.Option
			if cev.chainCtx.Err() != nil {
				return
			}
			if !reselect && livenessCheckCtx.Err() != nil {
				cev.logger.Warnf("Data plane is down")
				reselect = true
			}
			if reselect {
				cev.logger.Debugf("Reconnect with reselect")
				options = append(options, begin.WithReselect())
			}
			if err := <-cev.eventFactory.Request(options...); err == nil {
				return
			}
		}
	}
}

func (cev *eventLoop) monitorDataPlane() {
	ticker := time.NewTicker(cev.heal.livenessCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			deadlineCtx, deadlineCancel := context.WithDeadline(cev.chainCtx, time.Now().Add(cev.heal.livenessCheckTimeout))
			alive := cev.heal.livenessCheck(deadlineCtx, cev.conn)
			deadlineCancel()

			if !alive {
				return
			}
		case <-cev.eventLoopCtx.Done():
			return
		}
	}
}
