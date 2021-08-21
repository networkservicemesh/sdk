// Copyright (c) 2021 Cisco and/or its affiliates.
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

type key struct{}

func withEventFactory(parent context.Context, eventFactory EventFactory) context.Context {
	if parent.Value(key{}) != nil {
		return parent
	}
	ctx := context.WithValue(parent, key{}, eventFactory)
	return ctx
}

// FromContext - returns EventFactory from context
func FromContext(ctx context.Context) EventFactory {
	value := fromContext(ctx)
	if value == nil {
		panic("EventFactory not found please add begin chain element to your chain")
	}
	return value
}

func fromContext(ctx context.Context) EventFactory {
	value, ok := ctx.Value(key{}).(EventFactory)
	if ok {
		return value
	}
	return nil
}
