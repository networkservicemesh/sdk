// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

import (
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

// Option is authorization option for server
type Option interface {
	apply(*authorizePolicies)
}

// WithPolicies adds custom OPA policies
func WithPolicies(policies ...opa.AuthorizationPolicy) Option {
	return optionFunc(func(a *authorizePolicies) {
		a.policies = append(a.policies, policies...)
	})
}

// WithDefaultPolicies adds default OPA policies
func WithDefaultPolicies() Option {
	return optionFunc(func(a *authorizePolicies) {
		a.policies = append(
			a.policies,
			opa.WithAllTokensValidPolicy(),
			opa.WithTokensExpiredPolicy(),
			opa.WithTokenChainPolicy(),
			opa.WithLastTokenSignedPolicy(),
		)
	})
}

type optionFunc func(*authorizePolicies)

func (f optionFunc) apply(a *authorizePolicies) {
	f(a)
}
