// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package timeout provides a NetworkServiceServer chain element that times out expired connection
package timeout

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/expire"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type timeoutServer struct {
	ctx    context.Context
	timers expire.TimerMap
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections
//             for the subsequent chain elements.
// For the algorithm please see /pkg/tools/expire/README.md.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		ctx: ctx,
	}
}

func (s *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("timeoutServer", "Request")

	connID := request.GetConnection().GetId()

	// 1. Try to load and stop timer.
	t, loaded := s.timers.Load(connID)
	stopped := loaded && t.Stop()

	// 2. Send Request event.
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if stopped {
			// 2.1. Reset timer if Request event has failed and timer has been successfully stopped.
			t.Reset(clockTime.Until(t.ExpirationTime))
		}
		return nil, err
	}

	// 3. Delete the old timer.
	s.timers.Delete(connID)

	// 4. Create a new timer.
	t, err = s.newTimer(ctx, conn.Clone())
	if err != nil {
		// 4.1. If we have failed to create a new timer, Close the connection.
		if _, closeErr := next.Server(ctx).Close(ctx, conn); closeErr != nil {
			logger.Errorf("failed to close connection on error: %s %s", conn.GetId(), closeErr.Error())
		}
		return nil, err
	}

	// 5. Store timer.
	s.timers.Store(conn.GetId(), t)

	return conn, nil
}

func (s *timeoutServer) newTimer(ctx context.Context, conn *networkservice.Connection) (*expire.Timer, error) {
	clockTime := clock.FromContext(ctx)
	logger := log.FromContext(ctx).WithField("timeoutServer", "newTimer")

	// 1. Try to get executor from the context.
	executor := serializectx.GetExecutor(ctx, conn.GetId())
	if executor == nil {
		// 1.1. Fail if there is no executor.
		return nil, errors.Errorf("failed to get executor from context")
	}

	// 2. Compute expiration time.
	expirationTimestamp := conn.GetPrevPathSegment().GetExpires()
	if expirationTimestamp == nil {
		return nil, errors.Errorf("expiration for the previous path segment cannot be nil: %+v", conn)
	}
	expirationTime := expirationTimestamp.AsTime().Local()

	// 3. Create timer.
	var t *expire.Timer
	t = &expire.Timer{
		ExpirationTime: expirationTime,
		Timer: clockTime.AfterFunc(clockTime.Until(expirationTime), func() {
			// 3.1. All the timer action should be executed under the `executor.AsyncExec`.
			executor.AsyncExec(func() {
				// 3.2. Timer has probably been stopped and deleted or replaced with a new one, so we need to check it
				// before deleting.
				if tt, ok := s.timers.Load(conn.GetId()); !ok || tt != t {
					// 3.2.1. This timer has been stopped, nothing to do.
					return
				}

				// 3.3. Delete timer.
				s.timers.Delete(conn.GetId())

				// 3.4. Since `s.ctx` lives with the application, we need to create a new context with event scope
				// lifetime to prevent leaks.
				closeCtx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				// 3.5. Close timed out connection.
				if _, err := next.Server(ctx).Close(closeCtx, conn); err != nil {
					logger.Errorf("failed to close timed out connection: %s %s", conn.GetId(), err.Error())
				}
			})
		}),
	}

	return t, nil
}

func (s *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "Close")

	// 1. Check if we have a timer.
	t, ok := s.timers.LoadAndDelete(conn.GetId())
	if !ok {
		// 1.1. If there is no timer, there is nothing to do.
		logger.Warnf("connection has been already closed: %s", conn.GetId())
		return new(empty.Empty), nil
	}

	// 2. Stop it.
	t.Stop()

	// 3. Send Close event.
	return next.Server(ctx).Close(ctx, conn)
}
