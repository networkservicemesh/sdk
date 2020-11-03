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

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type timeoutServer struct {
	ctx       context.Context
	timers    timerMap
	executors executorMap
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

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	connID := request.GetConnection().GetId()

	newExecutor := new(serialize.Executor)
	// If we create an executor, we should be the first one who uses it, or else we can get a double Close issue:
	//     1. timeout close ->  : closing the Connection
	//     2. request ->        : creating `newExecutor`, storing into `executors`
	//     3. close ->          : locking `newExecutor`, checking `newExecutor == executors[connID]` - looks like
	//                            a retry Close case (but it isn't), double closing the Connection
	<-newExecutor.AsyncExec(func() {
		executor, loaded := t.executors.LoadOrStore(connID, newExecutor)
		if loaded {
			<-executor.AsyncExec(func() {
				// Executor has been possibly removed by `t.close()` at this moment, we need to store it back.
				exec, _ := t.executors.LoadOrStore(connID, executor)
				if exec != executor {
					// It can happen in such situation:
					//     1. -> timeout close  : locking `executor`
					//     2. -> request-1      : waiting on `executor`
					//     3. timeout close ->  : unlocking `executor`, removing it from `executors`
					//     4. -> request-2      : creating `exec`, storing into `executors`, locking `exec`
					//     5. -request-1->      : locking `executor`, trying to store it into `executors`
					// at 5. we get `request-1` locking `executor`, `request-2` locking `exec` and only `exec` stored
					// in `executors`. It means that `request-2` and all subsequent events will be executed in parallel
					// with `request-1`.
					err = errors.Errorf("race condition, parallel request execution: %v", connID)
					return
				}
				conn, err = t.request(ctx, request, executor)
			})
		} else {
			conn, err = t.request(ctx, request, executor)
		}
	})

	return conn, err
}

func (t *timeoutServer) request(ctx context.Context, request *networkservice.NetworkServiceRequest, executor *serialize.Executor) (*networkservice.Connection, error) {
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

	timer, err := t.createTimer(ctx, conn, executor)
	if err != nil {
		if _, closeErr := next.Server(ctx).Close(ctx, conn); closeErr != nil {
			err = errors.Wrapf(err, "error attempting to close failed connection %v: %+v", connID, closeErr)
		}
		return nil, err
	}

	t.timers.Store(connID, timer)

	return conn, nil
}

func (t *timeoutServer) createTimer(ctx context.Context, conn *networkservice.Connection, executor *serialize.Executor) (*time.Timer, error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "createTimer")

	expireTime, err := ptypes.Timestamp(conn.GetPath().GetPathSegments()[conn.GetPath().GetIndex()-1].GetExpires())
	if err != nil {
		return nil, err
	}

	conn = conn.Clone()

	timerPtr := new(*time.Timer)
	*timerPtr = time.AfterFunc(time.Until(expireTime), func() {
		executor.AsyncExec(func() {
			if timer, ok := t.timers.Load(conn.GetId()); !ok || timer != *timerPtr {
				logEntry.Warnf("timer has been already stopped: %v", conn.GetId())
				return
			}
			if err := t.close(t.ctx, conn, next.Server(ctx)); err != nil {
				logEntry.Errorf("failed to close timed out connection: %v %+v", conn.GetId(), err)
			}
		})
	})

	return *timerPtr, nil
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "Close")

	executor, ok := t.executors.Load(conn.GetId())
	if ok {
		<-executor.AsyncExec(func() {
			var exec *serialize.Executor
			if exec, ok = t.executors.Load(conn.GetId()); ok && exec == executor {
				err = t.close(ctx, conn, next.Server(ctx))
			} else {
				ok = false
			}
		})
	}
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", conn.GetId())
		return &empty.Empty{}, nil
	}

	return &empty.Empty{}, err
}

func (t *timeoutServer) close(ctx context.Context, conn *networkservice.Connection, nextServer networkservice.NetworkServiceServer) error {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "close")

	timer, ok := t.timers.LoadAndDelete(conn.GetId())
	if ok {
		timer.Stop()
	} else {
		// Last time we failed to close the Connection, let's do it again.
		logEntry.Warnf("retrying to close the connection: %v", conn.GetId())
	}

	_, err := nextServer.Close(ctx, conn)
	if err == nil {
		// If `nextServer.Close()` returns an error, the Connection is not truly closed, so we don't want to delete
		// the related executor.
		t.executors.Delete(conn.GetId())
	}

	return err
}
