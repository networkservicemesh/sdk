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
	"github.com/edwarnicke/serialize"
	"github.com/pkg/errors"
)

// CancellableExecutor is a cancellable serialize.Executor
type CancellableExecutor struct {
	executors *executorMap
	id        string
	executor  *serialize.Executor
}

// AsyncExec is same as serialize.Executor.AsyncExec() except that it returns error channel which sends error if
//           the executor is already canceled at the moment of execution
func (e *CancellableExecutor) AsyncExec(f func()) <-chan error {
	errCh := make(chan error, 1)
	e.executor.AsyncExec(func() {
		if executor, ok := e.executors.Load(e.id); !ok || executor != e.executor {
			errCh <- errors.Errorf("executor is already canceled: %v", e.id)
			return
		}
		f()
		close(errCh)
	})
	return errCh
}

// Cancel cancels the executor, should be called under the AsyncExec to prevent concurrent errors
func (e *CancellableExecutor) Cancel() {
	e.executors.Delete(e.id)
}
