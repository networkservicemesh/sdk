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

package setpayload

type configurable interface {
	setPayload(payload string)
}

// Option configures connect servers
type Option interface {
	apply(configurable)
}

type applyOptionFunc func(configurable)

func (a applyOptionFunc) apply(c configurable) {
	a(c)
}

// WithPayload sets default payload in case of empty payload
func WithPayload(payload string) Option {
	return applyOptionFunc(func(c configurable) {
		c.setPayload(payload)
	})
}
