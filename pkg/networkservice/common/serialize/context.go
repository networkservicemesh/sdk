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
	"context"
)

const (
	requestExecutorKey contextKeyType = "requestExecutor"
	closeExecutorKey   contextKeyType = "closeExecutor"
)

type contextKeyType string

func withExecutors(parent context.Context, requestExecutor, closeExecutor Executor) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(context.WithValue(parent,
		requestExecutorKey, requestExecutor),
		closeExecutorKey, closeExecutor)
}

// WithExecutorsFromContext wraps `parent` in a new context with the executors from `executorsContext`
func WithExecutorsFromContext(parent, executorsContext context.Context) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	if requestExecutor := RequestExecutor(executorsContext); requestExecutor != nil {
		parent = context.WithValue(parent, requestExecutorKey, requestExecutor)
	}
	if closeExecutor := CloseExecutor(executorsContext); closeExecutor != nil {
		parent = context.WithValue(parent, closeExecutorKey, closeExecutor)
	}
	return parent
}

// RequestExecutor returns Request `Executor`
func RequestExecutor(ctx context.Context) Executor {
	if executor, ok := ctx.Value(requestExecutorKey).(Executor); ok {
		return executor
	}
	return nil
}

// CloseExecutor returns Close `Executor`
func CloseExecutor(ctx context.Context) Executor {
	if executor, ok := ctx.Value(closeExecutorKey).(Executor); ok {
		return executor
	}
	return nil
}
