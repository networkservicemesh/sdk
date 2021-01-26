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

package refresh

import (
	"context"
	"time"
)

// Option is expire registry configuration option
type Option interface {
	apply(client *refreshNSEClient)
}

type applierFunc func(*refreshNSEClient)

func (f applierFunc) apply(c *refreshNSEClient) {
	f(c)
}

// WithDefaultExpiryDuration sets a default expiration_time if it is nil on NSE registration
func WithDefaultExpiryDuration(duration time.Duration) Option {
	return applierFunc(func(c *refreshNSEClient) {
		c.defaultExpiryDuration = duration
	})
}

// WithChainContext sets a chain context
func WithChainContext(ctx context.Context) Option {
	return applierFunc(func(c *refreshNSEClient) {
		c.chainContext = ctx
	})
}
