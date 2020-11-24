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
	"sync/atomic"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type serializer struct {
	executors map[string]*executor
	executor  serialize.Executor
	state     uint32
}

type executor struct {
	executor     serialize.Executor
	requestCount int
	// state can be accessed in several ways:
	// * executor is locked + requestCount is incremented : read, write without atomic
	// * executor is locked                               : read without atomic, write with atomic
	// * requestCount == 0                                : read with atomic
	state uint32
}

func (s *serializer) requestConnection(
	ctx context.Context,
	connID string,
	requestConn func(context.Context) (*networkservice.Connection, error),
) (conn *networkservice.Connection, err error) {
	var requestCh <-chan struct{}
	var exec *executor
	<-s.executor.AsyncExec(func() {
		var ok bool
		if exec, ok = s.executors[connID]; !ok {
			exec = new(executor)
			s.executors[connID] = exec
		}
		exec.requestCount++

		requestCh = exec.executor.AsyncExec(func() {
			for exec.state == 0 {
				exec.state = atomic.AddUint32(&s.state, 1)
			}
			conn, err = requestConn(withExecutors(ctx,
				newRequestExecutor(exec, connID),
				newCloseExecutor(exec, connID, func() {
					s.executor.AsyncExec(func() {
						s.clean(connID)
					})
				})))
			if err != nil {
				exec.state = 0
			}
		})
	})

	<-requestCh

	s.executor.AsyncExec(func() {
		exec.requestCount--
		s.clean(connID)
	})

	return conn, err
}

func (s *serializer) closeConnection(
	ctx context.Context,
	connID string,
	closeConn func(context.Context) (*empty.Empty, error),
) (_ *empty.Empty, err error) {
	logEntry := log.Entry(ctx).WithField("serializer", "closeConnection")

	var closeCh <-chan struct{}
	var exec *executor
	<-s.executor.AsyncExec(func() {
		var ok bool
		if exec, ok = s.executors[connID]; !ok {
			logEntry.Warnf("connection has been already closed: %v", connID)
			return
		}

		closeCh = exec.executor.AsyncExec(func() {
			if exec.state == 0 {
				logEntry.Warnf("connection has been already closed: %v", connID)
				return
			}
			_, err = closeConn(ctx)
			atomic.StoreUint32(&exec.state, 0)
		})
	})

	if exec != nil {
		<-closeCh

		s.executor.AsyncExec(func() {
			s.clean(connID)
		})
	}

	return &empty.Empty{}, err
}

func (s *serializer) clean(connID string) {
	exec, ok := s.executors[connID]
	if ok && exec.requestCount == 0 && atomic.LoadUint32(&exec.state) == 0 {
		delete(s.executors, connID)
	}
}
