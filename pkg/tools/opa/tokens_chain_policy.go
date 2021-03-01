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

// #nosec
const tokensPathCorrect = `
package policies

default token_chain_valid = false

token_chain_valid {
	count(input.path_segments) < 2
}

token_chain_valid {
	pair_count := count({x | input.path_segments[x]; pair_valid(input.path_segments[x].token, input.path_segments[x+1].token)})
	pair_count == count(input.path_segments) - 1
}

pair_valid(token1, token2) = r { 
	p1 := payload(token1)
	p2 := payload(token2)
	r := p1.aud == p2.sub
}

payload(token) = p {
    [_, p, _] := io.jwt.decode(token)
}
`

// WithTokenChainPolicy returns default policy for checking tokens chain in path
func WithTokenChainPolicy() *AuthorizationPolicy {
	return &AuthorizationPolicy{
		policySource: tokensPathCorrect,
		query:        "token_chain_valid",
		checker:      True("token_chain_valid"),
	}
}
