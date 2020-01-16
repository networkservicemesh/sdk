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

// Package serialize provides a simple means for Async or Sync execution of a func()
// with the guarantee that each func() will be executed exactly once and that all funcs()
// will be executed in order
package serialize

import "runtime"

// Executor - allows one at a time in order execution of func()s.  func()s can be queued Asynchronously or
// Synchronously
type Executor interface {
	AsyncExec(func())
	SyncExec(func())
}

type executor struct {
	execCh      chan func()
	finalizedCh chan struct{}
}

// NewExecutor - returns a new Executor
func NewExecutor() Executor {
	rv := &executor{
		execCh:      make(chan func(), 100),
		finalizedCh: make(chan struct{}),
	}
	go rv.eventLoop()
	runtime.SetFinalizer(rv, func(f *executor) {
		close(f.finalizedCh)
	})
	return rv
}

func (t *executor) eventLoop() {
	for {
		select {
		case exec := <-t.execCh:
			exec()
			continue
		case <-t.finalizedCh:
		}
		break
	}
}

// AsyncExec - queues exec for execution and returns without waiting for exec() to run
func (t *executor) AsyncExec(exec func()) {
	t.execCh <- exec
}

// SyncExec - queues exec for execution and returns *after* exec() has run
func (t *executor) SyncExec(exec func()) {
	done := make(chan struct{})
	t.execCh <- func() {
		exec()
		close(done)
	}
	<-done
}
