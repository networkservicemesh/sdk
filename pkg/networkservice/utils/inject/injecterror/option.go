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
	err                                error
	requestErrorTimes, closeErrorTimes []int
}

// Option is an option pattern for injectErrorClient/Server.
type Option func(o *options)

// WithError sets injectErrorClient/Server error.
func WithError(err error) Option {
	return func(o *options) {
		o.err = err
	}
}

// WithRequestErrorTimes sets injectErrorClient/Server request error times.
func WithRequestErrorTimes(failureTimes ...int) Option {
	return func(o *options) {
		o.requestErrorTimes = failureTimes
	}
}

// WithCloseErrorTimes sets injectErrorClient/Server close error times.
func WithCloseErrorTimes(failureTimes ...int) Option {
	return func(o *options) {
		o.closeErrorTimes = failureTimes
	}
}
