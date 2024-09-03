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

package injecterror

type options struct {
	err                                                      error
	registerErrorTimes, unregisterErrorTimes, findErrorTimes []int
}

// Option is an option pattern for injecterror package.
type Option func(o *options)

// WithError sets error.
func WithError(err error) Option {
	return func(o *options) {
		o.err = err
	}
}

// WithRegisterErrorTimes sets Register error times.
func WithRegisterErrorTimes(failureTimes ...int) Option {
	return func(o *options) {
		o.registerErrorTimes = failureTimes
	}
}

// WithFindErrorTimes sets Find error times.
func WithFindErrorTimes(failureTimes ...int) Option {
	return func(o *options) {
		o.findErrorTimes = failureTimes
	}
}

// WithUnregisterErrorTimes sets Unregister error times.
func WithUnregisterErrorTimes(failureTimes ...int) Option {
	return func(o *options) {
		o.unregisterErrorTimes = failureTimes
	}
}
