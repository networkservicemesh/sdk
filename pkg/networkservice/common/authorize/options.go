// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020-2021 Cisco Systems, Inc.
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

package authorize

// Option is authorization option for server
type Option interface {
	apply(*polyciesList)
}

// Any means authorize any request
func Any() Option {
	return WithPolicies(nil)
}

// WithPolicies sets custom policies
func WithPolicies(p ...Policy) Option {
	return optionFunc(func(l *polyciesList) {
		*l = p
	})
}

type optionFunc func(*polyciesList)

func (f optionFunc) apply(a *polyciesList) {
	f(a)
}
