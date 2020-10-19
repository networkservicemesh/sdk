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

// Package mapexecutor provides serialize.Executor with multiple task queues matched by IDs
package mapexecutor

import (
	"sync"

	"github.com/edwarnicke/serialize"
)

// Executor is a serialize.Executor with multiple task queues matched by IDs
type Executor struct {
	init      sync.Once
	executor  serialize.Executor
	executors map[string]*serialize.Executor
}

// AsyncExec starts task in the queue selected by the given ID (look at serialize.Executor)
func (e *Executor) AsyncExec(id string, f func()) <-chan struct{} {
	e.init.Do(func() {
		e.executors = map[string]*serialize.Executor{}
	})

	ready := make(chan struct{})
	chanPtr := new(<-chan struct{})
	e.executor.AsyncExec(func() {
		executor, ok := e.executors[id]
		if !ok {
			executor = &serialize.Executor{}
			e.executors[id] = executor
		}
		*chanPtr = executor.AsyncExec(f)
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
