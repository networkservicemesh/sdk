// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

// Package serializectx allows to set executor in the context
package serializectx

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

const (
	multiExecutorKey contextKeyType = "multiExecutor"
	executorKey      contextKeyType = "executor"
)

type contextKeyType string

// WithMultiExecutor wraps `parent` in a new context with multiexecutor.MultiExecutor
func WithMultiExecutor(parent context.Context, multiExecutor *multiexecutor.MultiExecutor) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, multiExecutorKey, multiExecutor)
}

// WithExecutor wraps `parent` in a new context with Executor
func WithExecutor(parent context.Context, executor *Executor) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, executorKey, executor)
}

// GetExecutor returns Executor
func GetExecutor(ctx context.Context, id string) *Executor {
	if executor, ok := ctx.Value(executorKey).(*Executor); ok && executor.id == id {
		return executor
	}
	if multiExecutor, ok := ctx.Value(multiExecutorKey).(*multiexecutor.MultiExecutor); ok {
		return &Executor{
			id: id,
			asyncExec: func(f func()) <-chan struct{} {
				return multiExecutor.AsyncExec(id, f)
			},
		}
	}
	return nil
}
