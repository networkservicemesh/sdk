// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// #nosec
const tokensValidPolicy = `
package test	

default tokens_valid = false
default index = 0

index = input.index

tokens_valid {	
	token := input.path_segments[index].token	
	cert := input.auth_info.certificate	
	io.jwt.verify_es256(token, cert) = true
}
`

// WithTokensValidPolicy returns default policy for checking tokens
func WithTokensValidPolicy() AuthorizationPolicy {
	return &authorizationPolicy{
		policySource: tokensValidPolicy,
		query:        "tokens_valid",
		checker:      True("tokens_valid"),
	}
}
