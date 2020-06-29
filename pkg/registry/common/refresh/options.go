// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

import "time"

type configurable interface {
	setRetryPeriod(time.Duration)
	setDefaultExpiration(duration time.Duration)
}

// Option is expire registry configuration option
type Option interface {
	apply(configurable)
}

type applierFunc func(configurable)

func (f applierFunc) apply(c configurable) {
	f(c)
}

// WithRetryPeriod sets a specific period to reconnect in case of a server returning an error
func WithRetryPeriod(duration time.Duration) Option {
	return applierFunc(func(c configurable) {
		c.setRetryPeriod(duration)
	})
}

// WithDefaultExpiration sets a default expiration to NSE if expiration was not set
func WithDefaultExpiration(duration time.Duration) Option {
	return applierFunc(func(c configurable) {
		c.setDefaultExpiration(duration)
	})
}
