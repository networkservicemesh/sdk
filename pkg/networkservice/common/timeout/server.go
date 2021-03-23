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
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

type timeoutServer struct {
	ctx    context.Context
	timers closeTimerMap
}

type closeTimer struct {
	expirationTime time.Time
	timer          clock.Timer
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections
//             for the subsequent chain elements.
// WARNING: `timeout` uses ctx as a context for the Close, so if there are any chain elements setting some data
//          in context in chain before the `timeout`, these changes won't appear in the Close context.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		ctx: ctx,
	}
}

func (s *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	clockTime := clock.FromContext(ctx)

	if err := s.validateRequest(ctx, request); err != nil {
		return nil, err
	}

	connID := request.GetConnection().GetId()

	t, loaded := s.timers.Load(connID)
	stopped := loaded && t.timer.Stop()

	expirationTime := request.GetConnection().GetPrevPathSegment().GetExpires().AsTime().Local()

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if stopped {
			t.timer.Reset(clockTime.Until(t.expirationTime))
		}
		return nil, err
	}

	s.timers.Store(connID, s.newTimer(ctx, expirationTime, conn.Clone()))

	return conn, nil
}

func (s *timeoutServer) validateRequest(ctx context.Context, request *networkservice.NetworkServiceRequest) error {
	conn := request.GetConnection()

	if conn.GetPrevPathSegment().GetExpires() == nil {
		return errors.Errorf("expiration for prev path segment cannot be nil. conn: %+v", request.GetConnection())
	}
	if serializectx.GetExecutor(ctx, conn.GetId()) == nil {
		return errors.New("no executor provided")
	}

	return nil
}

func (s *timeoutServer) newTimer(ctx context.Context, expirationTime time.Time, conn *networkservice.Connection) *closeTimer {
	logger := log.FromContext(ctx).WithField("timeoutServer", "newTimer")

	clockTime := clock.FromContext(ctx)

	tPtr := new(*closeTimer)
	*tPtr = &closeTimer{
		expirationTime: expirationTime,
		timer: clockTime.AfterFunc(clockTime.Until(expirationTime), func() {
			<-serializectx.GetExecutor(ctx, conn.GetId()).AsyncExec(func() {
				if t, ok := s.timers.LoadAndDelete(conn.GetId()); !ok || t != *tPtr {
					// this timer has been stopped
					return
				}

				closeCtx, cancel := context.WithCancel(s.ctx)
				defer cancel()

				if _, err := next.Server(ctx).Close(closeCtx, conn); err != nil {
					logger.Errorf("failed to close timed out connection: %s %s", conn.GetId(), err.Error())
				}
			})
		}),
	}

	return *tPtr
}

func (s *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logger := log.FromContext(ctx).WithField("timeoutServer", "Close")

	t, ok := s.timers.LoadAndDelete(conn.GetId())
	if !ok {
		logger.Warnf("connection has been already closed: %s", conn.GetId())
		return new(empty.Empty), nil
	}
	t.timer.Stop()

	return next.Server(ctx).Close(ctx, conn)
}
