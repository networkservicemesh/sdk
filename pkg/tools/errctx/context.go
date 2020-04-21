// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package errctx provides functions for stashing errors in context.Contexts
// Example of use:
// serverCtx, cancel := context.WithCancel(ctx)
// serverCtx = errctx.WithErr(ctx)
// go func() {
//		err = server.Serve(listener)
//		if err != nil {
//			select {
//			case <-ctx.Done():
//			default:
//				errctx.SetErr(serverCtx,err)
//				cancel()
//			}
//		}
//	}()
// <-serverCtx
// if err := errctx.Err(serverCtx); err != nil {
//    ...
// }
package errctx

import (
	"context"
)

type contextKeyType string

const (
	errorKey contextKeyType = "Error"
)

// Err - return the error (if any) stored by ListenAndServe in the context
func Err(ctx context.Context) error {
	if value := ctx.Value(errorKey); value != nil {
		return *value.(*error)
	}
	return nil
}

// WithErr - Returns a child context for which you can set an error
func WithErr(ctx context.Context) context.Context {
	var err error
	return context.WithValue(ctx, errorKey, err)
}

// SetErr - Set's an error.  Requires a previous call to WithErr
func SetErr(ctx context.Context, err error) {
	if value := ctx.Value(errorKey); value != nil {
		errPtr := value.(*error)
		*errPtr = err
	}
}
