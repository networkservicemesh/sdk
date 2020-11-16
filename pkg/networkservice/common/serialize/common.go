// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package serialize provides chain elements to make Request, Close processing in chain serial
package serialize

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func requestConnection(
	ctx context.Context,
	executors *executorMap,
	connID string,
	requestConn func(context.Context) (*networkservice.Connection, error),
) (conn *networkservice.Connection, err error) {
	// If we create an executor, we should be the first one who uses it, or else we can get a double Close issue:
	//     1. close ->    : closing the Connection
	//     2. request ->  : creating `newExecutor`, storing into `executors`
	//     3. close ->    : locking `newExecutor`, checking `newExecutor == executors[connID]` - looks like
	//                      a retry Close case (but it isn't), double closing the Connection
	newExecutor := new(serialize.Executor)
	<-newExecutor.AsyncExec(func() {
		for shouldRetry := true; shouldRetry; {
			shouldRetry = false

			executor, loaded := executors.LoadOrStore(connID, newExecutor)
			// We should set `requestExecutor`, `closeExecutor` into the request context so the following chain elements
			// can generate new Request, Close events and insert them into the chain in a serial way.
			requestExecutor := newRequestExecutor(executor, connID, executors)
			closeExecutor := newCloseExecutor(executor, connID, executors)
			if loaded {
				<-executor.AsyncExec(func() {
					// Executor has been possibly removed at this moment, we need to store it back.
					exec, _ := executors.LoadOrStore(connID, executor)
					if exec != executor {
						// It can happen in such situation:
						//     1. -> close      : locking `executor`
						//     2. -> request-1  : waiting on `executor`
						//     3. close ->      : unlocking `executor`, removing it from `executors`
						//     4. -> request-2  : creating `exec`, storing into `executors`, locking `exec`
						//     5. -request-1->  : locking `executor`, trying to store it into `executors`
						// at 5. we get `request-1` locking `executor`, `request-2` locking `exec` and only `exec` stored
						// in `executors`. It means that `request-2` and all subsequent events will be executed in parallel
						// with `request-1`.
						shouldRetry = true
						return
					}
					if conn, err = requestConn(withExecutors(ctx, requestExecutor, closeExecutor)); err != nil {
						executors.Delete(connID)
					}
				})
			} else if conn, err = requestConn(withExecutors(ctx, requestExecutor, closeExecutor)); err != nil {
				executors.Delete(connID)
			}
		}
	})

	return conn, err
}

func closeConnection(
	ctx context.Context,
	executors *executorMap,
	connID string,
	closeConn func(context.Context) (*empty.Empty, error),
) (_ *empty.Empty, err error) {
	logEntry := log.Entry(ctx).WithField("serialize", "close")

	executor, ok := executors.Load(connID)
	if ok {
		<-executor.AsyncExec(func() {
			var exec *serialize.Executor
			if exec, ok = executors.Load(connID); ok && exec == executor {
				// We don't set `requestExecutor`, `closeExecutor` into the close context because they should be
				// canceled by the time anything can be executed with them.
				_, err = closeConn(ctx)
				executors.Delete(connID)
			} else {
				ok = false
			}
		})
	}
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", connID)
		return &empty.Empty{}, nil
	}

	return &empty.Empty{}, err
}
