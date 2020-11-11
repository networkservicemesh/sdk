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
	executorKey contextKeyType = "serialize.Executor"
)

type contextKeyType string

// WithExecutor wraps `parent` in a new context with the CancellableExecutor
func WithExecutor(parent context.Context, executor *CancellableExecutor) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, executorKey, executor)
}

// Executor returns CancellableExecutor
func Executor(ctx context.Context) *CancellableExecutor {
	if executor, ok := ctx.Value(executorKey).(*CancellableExecutor); ok {
		return executor
	}
	return nil
}
