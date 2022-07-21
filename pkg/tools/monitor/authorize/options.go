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

import "github.com/networkservicemesh/sdk/pkg/tools/spire"

type options struct {
	policies              policiesList
	spiffeIDConnectionMap *spire.SpiffeIDConnectionMap
}

// Option is authorization option for monitor connection server
type Option func(*options)

// Any authorizes any call of request/close
func Any() Option {
	return WithPolicies(nil)
}

// WithPolicies sets custom policies
func WithPolicies(p ...Policy) Option {
	return func(o *options) {
		o.policies = p
	}
}

// WithSpiffeIDConnectionMap sets map to keep spiffeIDConnectionMap to authorize connections with MonitorServer
func WithSpiffeIDConnectionMap(s *spire.SpiffeIDConnectionMap) Option {
	return func(o *options) {
		o.spiffeIDConnectionMap = s
	}
}
