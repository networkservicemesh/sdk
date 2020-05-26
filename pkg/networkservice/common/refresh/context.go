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

// Package refresh allows the setting of a refresh context in the context of the request
package refresh

import (
	"context"
)

const (
	refreshContextKey contextKeyType = "RefreshContext"
)

type contextKeyType string

// withRefreshContext -
//    Wraps 'parent' in a new Context that has the RefreshContext
func withRefreshContext(parent, refreshContext context.Context) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, refreshContextKey, refreshContext)
}

// refreshContext -
//   Returns the RefreshContext
func refreshContext(ctx context.Context) context.Context {
	if rv, ok := ctx.Value(refreshContextKey).(context.Context); ok {
		return rv
	}
	return nil
}
