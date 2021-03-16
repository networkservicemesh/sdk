// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package serializectx

import "github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"

// Executor is a wrapper around `serialize.Executor.AsyncExec` + ID
type Executor struct {
	id        string
	asyncExec func(f func()) <-chan struct{}
}

// NewExecutor creates a new Executor from given multiexecutor.MultiExecutor and ID
func NewExecutor(multiExecutor *multiexecutor.MultiExecutor, id string) *Executor {
	return &Executor{
		id: id,
		asyncExec: func(f func()) <-chan struct{} {
			return multiExecutor.AsyncExec(id, f)
		},
	}
}

// AsyncExec is a `serialize.Executor.AsyncExec`
func (e *Executor) AsyncExec(f func()) <-chan struct{} {
	return e.asyncExec(f)
}
