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

// Package refresh allows the setting of the refresh context into the context of the request.
// Refresh context is used to distinguish requests with the same connectionID (repeating refresh requests will have
// RefreshContext while initial request and new requests from healClient will not) and to be able
// to guarantee refresh canceling by canceling it's refresh context
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

// RefreshContext -
//   Returns the RefreshContext
func RefreshContext(ctx context.Context) context.Context {
	if rv, ok := ctx.Value(refreshContextKey).(context.Context); ok {
		return rv
	}
	return nil
}

// TODO remove it
const refreshNumberKey contextKeyType = "RefreshNumber"

type RefreshNumber struct {
	Number int
}

func withRefreshNumber(parent context.Context, refreshNumber *RefreshNumber) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, refreshNumberKey, refreshNumber)
}

func GetRefreshNumber(ctx context.Context) *RefreshNumber {
	if rv, ok := ctx.Value(refreshNumberKey).(*RefreshNumber); ok {
		return rv
	}
	return nil
}
