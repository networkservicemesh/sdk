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
	"sync/atomic"

	"github.com/pkg/errors"
)

// Executor is same as serialize.Executor except that it returns error channel
type Executor interface {
	AsyncExec(f func() error) <-chan error
}

type executorFunc func(f func() error) <-chan error

func (ef executorFunc) AsyncExec(f func() error) <-chan error {
	return ef(f)
}

func newRequestExecutor(exec *executor, id string) Executor {
	return newExecutor(exec, exec.state, id)
}

func newCloseExecutor(exec *executor, id string, clean func()) Executor {
	state := exec.state
	return executorFunc(func(f func() error) <-chan error {
		executor := newExecutor(exec, state, id)
		return executor.AsyncExec(func() error {
			if err := f(); err != nil {
				return err
			}
			atomic.StoreUint32(&exec.state, 0)
			clean()
			return nil
		})
	})
}

func newExecutor(exec *executor, state uint32, id string) Executor {
	return executorFunc(func(f func() error) <-chan error {
		errCh := make(chan error, 1)
		exec.executor.AsyncExec(func() {
			if exec.state != state {
				errCh <- errors.Errorf("connection is already closed: %v", id)
				return
			}
			errCh <- f()
			close(errCh)
		})
		return errCh
	})
}
