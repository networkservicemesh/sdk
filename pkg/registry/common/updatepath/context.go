// Copyright (c) 2024 Cisco and/or its affiliates.
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

package updatepath

import (
	"context"
	"time"
)

type key struct{}

// ExpirationTimeFromContext returns the expiration time stored in context.
func ExpirationTimeFromContext(ctx context.Context) *time.Time {
	if value, ok := ctx.Value(key{}).(*time.Time); ok {
		return value
	}
	return nil
}

// withExpirationTime sets the expiration time stored in context.
func withExpirationTime(ctx context.Context, t *time.Time) context.Context {
	return context.WithValue(ctx, key{}, t)
}
