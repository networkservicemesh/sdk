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

// Package multiexecutor provides serialize.Executor with multiple task queues matched by IDs
package multiexecutor

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/edwarnicke/serialize"
)

const (
	cleanupTimeout = 10 * time.Millisecond
)

// Executor is a serialize.Executor with multiple task queues matched by IDs
type Executor struct {
	executor  serialize.Executor
	executors map[string]*executor
	updated   int32
}

type executor struct {
	executor serialize.Executor
	count    int32
}

func NewExecutor(ctx context.Context) *Executor {
	e := &Executor{
		executors: map[string]*executor{},
	}

	go e.cleanup(ctx)

	return e
}

func (e *Executor) cleanup(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(cleanupTimeout):
			<-e.executor.AsyncExec(func() {
				if atomic.CompareAndSwapInt32(&e.updated, 1, 0) {
					return
				}
				for id, ex := range e.executors {
					if atomic.LoadInt32(&ex.count) == 0 {
						delete(e.executors, id)
					}
				}
			})
		}
	}
}

// AsyncExec starts task in the queue selected by the given ID (look at serialize.Executor)
func (e *Executor) AsyncExec(id string, f func()) <-chan struct{} {
	ready := make(chan struct{})
	chanPtr := new(<-chan struct{})
	e.executor.AsyncExec(func() {
		ex, ok := e.executors[id]
		if !ok {
			ex = &executor{}
			e.executors[id] = ex
		}
		atomic.AddInt32(&ex.count, 1)

		*chanPtr = ex.executor.AsyncExec(func() {
			f()
			atomic.AddInt32(&ex.count, -1)
			atomic.CompareAndSwapInt32(&e.updated, 0, 1)
		})
		close(ready)
	})

	return wrapChanPtr(ready, chanPtr)
}

func wrapChanPtr(ready <-chan struct{}, chanPtr *<-chan struct{}) <-chan struct{} {
	rv := make(chan struct{})
	go func() {
		<-ready
		<-*chanPtr
		close(rv)
	}()
	return rv
}
