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

import (
	"sync"
	"sync/atomic"
)

const (
	channelSize = 100
)

// Executor - a struct that can be used to guarantee exclusive, in order execution of functions.
type Executor struct {
	execCh chan func()
	init   sync.Once
	count  int32
}

// NewExecutor - provides a new Executor
// Deprecated: Please just used Executor{} in the future.  The zero value of Executor works just fine.
func NewExecutor() Executor {
	return Executor{}
}

// AsyncExec - guarantees f() will be executed Exclusively and in the Order submitted.
//        It immediately returns a channel that will be closed when f() has completed execution.
func (e *Executor) AsyncExec(f func()) <-chan struct{} {
	// Initialize *once*
	e.init.Do(func() {
		e.execCh = make(chan func(), channelSize)
	})
	// Start go routine if we don't have one
	if atomic.AddInt32(&e.count, 1) == 1 {
		go func() {
			for g := range e.execCh {
				g()
				if atomic.AddInt32(&e.count, -1) == 0 {
					return
				}
			}
		}()
	}
	done := make(chan struct{})
	e.execCh <- func() {
		f()
		close(done)
	}
	return done
}
