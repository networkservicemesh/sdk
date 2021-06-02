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

package heal

import (
	"context"
)

const (
	nseFindHealingKey contextKeyType = "nseFindHealing"
	nsFindHealingKey  contextKeyType = "nsFindHealing"
)

type contextKeyType string

func withNSEFindHealing(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nseFindHealingKey, true)
}

func isNSEFindHealing(ctx context.Context) bool {
	if _, ok := ctx.Value(nseFindHealingKey).(bool); ok {
		return true
	}
	return false
}

func withNSFindHealing(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, nsFindHealingKey, true)
}

func isNSFindHealing(ctx context.Context) bool {
	if _, ok := ctx.Value(nsFindHealingKey).(bool); ok {
		return true
	}
	return false
}
