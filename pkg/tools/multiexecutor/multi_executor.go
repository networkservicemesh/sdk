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

// Package multiexecutor provides serial executor with multiple execution queues
package multiexecutor

import (
	"sync"

	"github.com/edwarnicke/serialize"
)

// Executor is a serial executor with multiple execution queues
type Executor struct {
	executors map[string]*executor
	executor  serialize.Executor
	once      sync.Once
}

type executor struct {
	executor serialize.Executor
	refCount int
}

// AsyncExec executes `f` serially in the `id` queue
func (e *Executor) AsyncExec(id string, f func()) (ch <-chan struct{}) {
	e.once.Do(func() {
		e.executors = make(map[string]*executor)
	})

	<-e.executor.AsyncExec(func() {
		exec, ok := e.executors[id]
		if !ok {
			exec = new(executor)
			e.executors[id] = exec
		}
		exec.refCount++

		ch = exec.executor.AsyncExec(func() {
			f()
			e.executor.AsyncExec(func() {
				exec.refCount--
				if exec.refCount == 0 {
					delete(e.executors, id)
				}
			})
		})
	})
	return ch
}
