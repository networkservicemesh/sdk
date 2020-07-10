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

// Package capturecontext provides a chain element that adds into context reference to current context for passing it
// from the adapter server/client to the next client/server to avoid the problem with losing values from adapted server/client context.
package capturecontext

import (
	"context"
)

type contextKeyType string

const capturedContext contextKeyType = "capturedContext"

// WithCapturedContext - adds record with nil reference into context or nullifies it
func WithCapturedContext(ctx context.Context) context.Context {
	if v := ctx.Value(capturedContext); v != nil {
		if b, ok := v.(*context.Context); ok {
			*b = nil
			return ctx
		}
	}
	var d context.Context = nil
	return context.WithValue(ctx, capturedContext, &d)
}

// IsCapturedContext - checks if the context record has a non-zero reference
func IsCapturedContext(ctx context.Context) bool {
	val, ok := ctx.Value(capturedContext).(*context.Context)
	return ok && *val != nil
}

// CapturedContext - returns previously written context
func CapturedContext(ctx context.Context) context.Context {
	if val, ok := ctx.Value(capturedContext).(*context.Context); ok {
		return *val
	}
	return nil
}

// captureContext - adds reference on current context in context record
func captureContext(ctx context.Context) {
	if ctx == nil {
		return
	}
	if val, ok := ctx.Value(capturedContext).(*context.Context); ok {
		*val = ctx
	}
}
