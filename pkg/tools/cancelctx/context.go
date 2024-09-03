// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package cancelctx provides methods for creating context with injected cancel func for it
package cancelctx

import "context"

const (
	cancelKey contextKeyType = "cancel"
)

type contextKeyType string

// WithCancel creates a context with cancel func and injects it into the context.
func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	ctx, cancel := context.WithCancel(parent)
	ctx = context.WithValue(ctx, cancelKey, cancel)

	return ctx, cancel
}

// FromContext returns a cancel func for the context.
func FromContext(ctx context.Context) context.CancelFunc {
	if cancel, ok := ctx.Value(cancelKey).(context.CancelFunc); ok {
		return cancel
	}
	return nil
}
