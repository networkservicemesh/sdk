// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
)

type option struct {
	cancelCtx     context.Context
	extendContext context.Context
}

// Option - event option.
type Option func(*option)

// CancelContext - optionally provide a context that, when canceled will preclude the event from running.
func CancelContext(cancelCtx context.Context) Option {
	return func(o *option) {
		o.cancelCtx = cancelCtx
	}
}

// ExtendContext - optionally provides a context which extends factory's context with its values.
func ExtendContext(extendCountext context.Context) Option {
	return func(o *option) {
		o.extendContext = extendCountext
	}
}
