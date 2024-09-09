// Copyright (c) 2020-2023 Cisco Systems, Inc.
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

// Package extend allows you to extend a context with values from another context
package extend

import (
	"context"
)

type extendedContext struct {
	context.Context
	valuesContext context.Context
}

func (ec *extendedContext) Value(key interface{}) interface{} {
	return ec.valuesContext.Value(key)
}

type joinedValuesContext struct {
	context.Context
	valuesContext context.Context
}

func (ec *joinedValuesContext) Value(key interface{}) interface{} {
	val := ec.valuesContext.Value(key)
	if val != nil {
		return val
	}
	return ec.Context.Value(key)
}

// WithValuesFromContext - creates a child context with the Values from valuesContext rather than the parent.
func WithValuesFromContext(parent, valuesContext context.Context) context.Context {
	return &extendedContext{
		Context:       parent,
		valuesContext: valuesContext,
	}
}

// WithBothValuesFromContext - creates a child context with the Values from both parent and values Contexts.
func WithBothValuesFromContext(parent, valuesContext context.Context) context.Context {
	return &joinedValuesContext{
		Context:       parent,
		valuesContext: valuesContext,
	}
}
