// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package opa

import _ "embed"

//go:embed policies/tokens_valid.rego
var tokensValidPolicySource string

//go:embed policies/prev_token_signed.rego
var prevTokenSignedPolicySource string

//go:embed policies/next_token_signed.rego
var currTokenSignedPolicySource string

//go:embed policies/tokens_chained.rego
var tokensChainedPolicySource string

//go:embed policies/tokens_expired.rego
var tokensExpiredPolicySource string

//go:embed policies/service_connection.rego
var tokensServiceConnectionPolicySource string

// WithTokensValidPolicy returns default policy for checking that all tokens in the path can be decoded.
func WithTokensValidPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensValidPolicySource,
		query:        "tokens_valid",
		checker:      True("tokens_valid"),
	}
}

// WithNextTokenSignedPolicy returns default policy for checking that last token in path is signed.
func WithNextTokenSignedPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: currTokenSignedPolicySource,
		query:        "next_token_signed",
		checker:      True("next_token_signed"),
	}
}

// WithPrevTokenSignedPolicy returns default policy for checking that last token in path is signed.
func WithPrevTokenSignedPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: prevTokenSignedPolicySource,
		query:        "prev_token_signed",
		checker:      True("prev_token_signed"),
	}
}

// WithTokenChainPolicy returns default policy for checking tokens chain in path
func WithTokenChainPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensChainedPolicySource,
		query:        "tokens_chained",
		checker:      True("tokens_chained"),
	}
}

// WithTokensExpiredPolicy returns default policy for checking tokens expiration
func WithTokensExpiredPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensExpiredPolicySource,
		query:        "tokens_expired",
		checker:      False("tokens_expired"),
	}
}

func WithMonitorConnectionServerPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensServiceConnectionPolicySource,
		query:        "service_connection",
		checker:      True("service_connection"),
	}
}
