// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type timeoutServer struct {
	ctx    context.Context
	timers timerMap
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

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "request")

	connID := request.GetConnection().GetId()

	if timer, ok := t.timers.LoadAndDelete(connID); ok {
		if !timer.Stop() {
			// Even if we failed to stop the timer, we should execute. It does mean that the timeout action
			// is waiting on `executor.AsyncExec()` until we will finish.
			// Since timer is being deleted under the `executor.AsyncExec()` this can't be a situation when
			// the Request is executing after the timeout Close. Such case cannot be distinguished with the
			// first-request case.
			logEntry.Warnf("connection has been timed out, re requesting: %v", connID)
		}
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	timer, err := t.createTimer(ctx, conn)
	if err != nil {
		if _, closeErr := next.Server(ctx).Close(ctx, conn); closeErr != nil {
			err = errors.Wrapf(err, "error attempting to close failed connection %v: %+v", connID, closeErr)
		}
		return nil, err
	}

	t.timers.Store(connID, timer)

	return conn, nil
}

func (t *timeoutServer) createTimer(ctx context.Context, conn *networkservice.Connection) (*time.Timer, error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "createTimer")

	executor := serialize.GetExecutor(ctx)
	if executor == nil {
		return nil, errors.New("no executor provided")
	}

	if conn.GetPrevPathSegment().GetExpires() == nil {
		return nil, errors.Errorf("expiration for prev path segment cannot be nil. conn: %+v", conn)
	}
	expireTime, err := ptypes.Timestamp(conn.GetPrevPathSegment().GetExpires())
	if err != nil {
		return nil, err
	}

	conn = conn.Clone()

	timerPtr := new(*time.Timer)
	*timerPtr = time.AfterFunc(time.Until(expireTime), func() {
		<-executor.AsyncExec(func() {
			if timer, _ := t.timers.Load(conn.GetId()); timer != *timerPtr {
				logEntry.Warnf("timer has been already stopped: %v", conn.GetId())
				return
			}
			t.timers.Delete(conn.GetId())
			if _, err := next.Server(ctx).Close(t.ctx, conn); err != nil {
				logEntry.Errorf("failed to close timed out connection: %v %+v", conn.GetId(), err)
			}
		})
	})

	return *timerPtr, nil
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "createTimer")

	timer, ok := t.timers.LoadAndDelete(conn.GetId())
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", conn.GetId())
		return new(empty.Empty), nil
	}
	timer.Stop()

	return next.Server(ctx).Close(ctx, conn)
}
