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

// Executor is same as serialize.Executor except that it returns error channel
type Executor interface {
	AsyncExec(f func()) <-chan error
}

type executorFunc func(f func()) <-chan error

func (ef executorFunc) AsyncExec(f func()) <-chan error {
	return ef(f)
}

func newRequestExecutor(executor *serialize.Executor, id string, executors *executorMap) Executor {
	return executorFunc(func(f func()) <-chan error {
		errCh := make(chan error, 1)
		executor.AsyncExec(func() {
			if exec, ok := executors.Load(id); !ok || exec != executor {
				errCh <- errors.Errorf("connection is already closed: %v", id)
				return
			}
			f()
			close(errCh)
		})
		return errCh
	})
}

func newCloseExecutor(executor *serialize.Executor, id string, executors *executorMap) Executor {
	return executorFunc(func(f func()) <-chan error {
		exec := newRequestExecutor(executor, id, executors)
		return exec.AsyncExec(func() {
			f()
			executors.Delete(id)
		})
	})
}
