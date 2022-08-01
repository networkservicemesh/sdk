// Copyright (c) 2022 Cisco and/or its affiliates.
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

type options struct {
	registerPolicies     policiesList
	unregisterPolicies   policiesList
	spiffeIDResourcesMap *spiffeIDResourcesMap
}

// Option is authorization option for server
type Option func(*options)

// Any authorizes any call of request/close
func Any() Option {
	return func(o *options) {
		o.registerPolicies = nil
		o.unregisterPolicies = nil
	}
}

// WithRegisterPolicies sets custom policies for register check
func WithRegisterPolicies(p ...Policy) Option {
	return func(o *options) {
		o.registerPolicies = p
	}
}

// WithUnregisterPolicies sets custom policies for unregister check
func WithUnregisterPolicies(p ...Policy) Option {
	return func(o *options) {
		o.unregisterPolicies = p
	}
}

func WithSpiffeIDNSEsMap(m *spiffeIDResourcesMap) Option {
	return func(o *options) {
		o.spiffeIDResourcesMap = m
	}
}
