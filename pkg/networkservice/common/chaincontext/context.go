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

package chaincontext

import (
	"context"
)

const (
	chainContextKey contextKeyType = "ChainContext"
)

type contextKeyType string

// WithChainContext -
//    Wraps 'parent' per-request Context in a new Context that has the 'chain' Context
//    Returns wrapped per-request context
func WithChainContext(parent, chain context.Context) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, chainContextKey, chain)
}

// ChainContext -
//   Returns the ChainContext
func ChainContext(ctx context.Context) context.Context {
	if rv, ok := ctx.Value(chainContextKey).(context.Context); ok {
		return rv
	}
	return nil
}
