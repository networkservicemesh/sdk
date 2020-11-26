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

import "github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"

// ExecutorFunc is a serialize.Executor.AsyncExec func type
type ExecutorFunc func(f func()) <-chan struct{}

func newExecutorFunc(id string, executor *multiexecutor.Executor) ExecutorFunc {
	return func(f func()) <-chan struct{} {
		return executor.AsyncExec(id, f)
	}
}

// AsyncExec calls ExecutorFunc
func (e ExecutorFunc) AsyncExec(f func()) <-chan struct{} {
	return e(f)
}
