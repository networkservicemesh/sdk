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

package serialize

import (
	"sync"

	"github.com/edwarnicke/serialize"
)

type multiExecutor struct {
	executors map[string]*refCountExecutor
	executor  serialize.Executor
	once      sync.Once
}

type refCountExecutor struct {
	count    int
	executor serialize.Executor
}

func (e *multiExecutor) AsyncExec(id string, f func()) (ch <-chan struct{}) {
	e.once.Do(func() {
		e.executors = make(map[string]*refCountExecutor)
	})

	<-e.executor.AsyncExec(func() {
		exec, ok := e.executors[id]
		if !ok {
			exec = new(refCountExecutor)
			e.executors[id] = exec
		}
		exec.count++

		ch = exec.executor.AsyncExec(func() {
			f()
			e.executor.AsyncExec(func() {
				exec.count--
				if exec.count == 0 {
					delete(e.executors, id)
				}
			})
		})
	})
	return ch
}

func (e *multiExecutor) Executor(id string) Executor {
	return executorFunc(func(f func()) <-chan struct{} {
		return e.AsyncExec(id, f)
	})
}
