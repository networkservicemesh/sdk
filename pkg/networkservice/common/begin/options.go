// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

package begin

import (
	"context"
	"time"
)

type option struct {
	cancelCtx      context.Context
	reselect       bool
	contextTimeout time.Duration
	reselectFunc   ReselectFunc
}

// Option - event option
type Option func(*option)

// CancelContext - optionally provide a context that, when canceled will preclude the event from running
func CancelContext(cancelCtx context.Context) Option {
	return func(o *option) {
		o.cancelCtx = cancelCtx
	}
}

// WithReselect - optionally clear Mechanism and NetworkServiceName to force reselect
func WithReselect() Option {
	return func(o *option) {
		o.reselect = true
	}
}

// WithContextTimeout - set a custom timeout for a context in begin.Close
func WithContextTimeout(timeout time.Duration) Option {
	return func(o *option) {
		o.contextTimeout = timeout
	}
}

func WithReselectFunc(reselectFunc ReselectFunc) Option {
	return func(o *option) {
		o.reselectFunc = reselectFunc
	}
}
