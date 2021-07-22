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

// Package closectx provides helper method to create Close/Unregister context in case of Request/Register failure
package closectx

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

const closeTimeout = 15 * time.Second

// New creates a new context for Close/Unregister in case of Request/Register failure
func New(parent, valuesCtx context.Context) (ctx context.Context, cancel context.CancelFunc) {
	clockTime := clock.FromContext(valuesCtx)

	var timeout time.Duration
	if deadline, ok := valuesCtx.Deadline(); ok {
		timeout = clockTime.Until(deadline)
	}
	if timeout < closeTimeout {
		timeout = closeTimeout
	}

	ctx = extend.WithValuesFromContext(parent, valuesCtx)

	return clockTime.WithTimeout(ctx, timeout)
}
