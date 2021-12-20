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

package retry

import "time"

type configurable interface {
	setInterval(time.Duration)
	setTryTimeout(time.Duration)
}

// Option is NS/NSE retry client config option
type Option interface {
	apply(configurable)
}

type applier func(configurable)

func (f applier) apply(c configurable) {
	f(c)
}

// WithTryTimeout sets timeout for the request and close operations try.
func WithTryTimeout(tryTimeout time.Duration) Option {
	return applier(func(c configurable) {
		c.setTryTimeout(tryTimeout)
	})
}

// WithInterval sets delay interval before next try.
func WithInterval(interval time.Duration) Option {
	return applier(func(c configurable) {
		c.setInterval(interval)
	})
}
