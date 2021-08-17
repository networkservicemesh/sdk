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

// Package postpone is used to create a context with postponed deadline for some cleanup operations.
package postpone

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

// Context returns a function providing the context with the same timeout as ctx has at this moment.
func Context(ctx context.Context) func() (context.Context, context.CancelFunc) {
	deadline, ok := ctx.Deadline()
	if !ok {
		return func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		}
	}

	clockTime := clock.FromContext(ctx)
	timeout := clockTime.Until(deadline)

	return func() (context.Context, context.CancelFunc) {
		return clockTime.WithTimeout(context.Background(), timeout)
	}
}

// ContextWithValues is the same as a Context, but also provided context has the same values as ctx.
func ContextWithValues(ctx context.Context) func() (context.Context, context.CancelFunc) {
	ctxFunc := Context(ctx)
	return func() (context.Context, context.CancelFunc) {
		postponedCtx, cancel := ctxFunc()
		return extend.WithValuesFromContext(postponedCtx, ctx), cancel
	}
}
