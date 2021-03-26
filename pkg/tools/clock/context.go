// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

package clock

import (
	"context"
)

const (
	clockKey contextKeyType = "Clock"
)

type contextKeyType string

var clock = new(clockImpl)

// WithClock wraps parent in a new context with Clock
func WithClock(parent context.Context, clock Clock) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, clockKey, clock)
}

// FromContext returns Clock from context
func FromContext(ctx context.Context) Clock {
	if rv, ok := ctx.Value(clockKey).(Clock); ok {
		return rv
	}
	return clock
}
