// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

//go:embed policies/tokens_valid.opa
var tokensValidPolicySource string

//go:embed policies/prev_token_signed.opa
var prevTokenSignedPolicySource string

//go:embed policies/curr_token_signed.opa
var currTokenSignedPolicySource string

//go:embed policies/tokens_chained.opa
var tokensChainedPolicySource string

//go:embed policies/tokens_expired.opa
var tokensExpiredPolicySource string

// WithTokensValidPolicy returns default policy for checking that all tokens in the path can be decoded.
func WithTokensValidPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensValidPolicySource,
		query:        "tokens_valid",
		checker:      True("tokens_valid"),
	}
}

// WithCurrentTokenSignedPolicy returns default policy for checking that last token in path is signed.
func WithCurrentTokenSignedPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: currTokenSignedPolicySource,
		query:        "curr_token_signed",
		checker:      True("curr_token_signed"),
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
