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

package opa_test

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serviceSample struct {
	tokenGenerator token.GeneratorFunc
	name           string
}

type chainSample struct {
	name         string
	services     []serviceSample
	isValidChain bool
}

func getSamples() []chainSample {
	return []chainSample{
		{
			name: "Valid chain",
			services: []serviceSample{
				{
					name: "nsc",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "nsc",
						Audience: []string{"nsmgr1"},
					}),
				},
				{
					name: "nsmgr1",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "nsmgr1",
						Audience: []string{"nsmgr2"},
					}),
				},
				{
					name: "nsmgr2",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "nsmgr2",
						Audience: []string{"nse"},
					}),
				},
			},
			isValidChain: true,
		},
		{
			name: "Invalid chain",
			services: []serviceSample{
				{
					name: "nsc",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "nsc",
						Audience: []string{"nsmgr1"},
					}),
				},
				{
					name: "nsmgr1",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "nsmgr1",
						Audience: []string{"nsmgr2"},
					}),
				},
				{
					name: "nsmgr2",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject:  "spy",
						Audience: []string{"nse"},
					}),
				},
				{
					name: "nse",
					tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
						Subject: "nse",
					}),
				},
			},
			isValidChain: false,
		},
	}
}

func TestWithTokensPathValidPolicy(t *testing.T) {
	p, err := opa.PolicyPath("policies/tokens_chained.rego").Read()
	require.NoError(t, err)

	samples := getSamples()

	for i := 0; i < len(samples); i++ {
		conn := &networkservice.Connection{
			Path: &networkservice.Path{},
		}

		sample := &samples[i]

		for _, srvc := range sample.services {
			tok, expire, err := srvc.tokenGenerator(nil)
			require.NoError(t, err)
			conn.Path.PathSegments = append(conn.Path.PathSegments, &networkservice.PathSegment{
				Name:    srvc.name,
				Token:   tok,
				Expires: timestamppb.New(expire),
			})
		}

		if sample.isValidChain {
			require.NoError(t, p.Check(context.Background(), conn.GetPath()), sample.name)
		} else {
			require.Error(t, p.Check(context.Background(), conn.GetPath()), sample.name)
		}
	}
}
