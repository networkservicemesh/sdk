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


package tests

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/tools/opautils"

	"github.com/dgrijalva/jwt-go"
	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestTokensMatchingPolicy(t *testing.T){
	suits := []struct {
		name  string
		tokens []string
		isMatching bool
	}{
		{
			name:  "simple positive test with two tokens",
			tokens: []string{
				generateTokenWithClaims(&jwt.StandardClaims{
					Audience:  "nsmgr1",
					Subject:   "nsc",
				}),
				generateTokenWithClaims(&jwt.StandardClaims{
					Audience:  "nsmgr2",
					Subject:   "nsmgr1",
				}),
			},
			isMatching: true,
		},
		{
			name:    "negative test with not matches tokens",
			tokens: []string{
				generateTokenWithClaims(&jwt.StandardClaims{
					Audience:  "nsmgr1",
					Subject:   "nsc",
				}),
				generateTokenWithClaims(&jwt.StandardClaims{
					Audience:  "nsmgr2",
					Subject:   "nsmgr1",
				}),
				generateTokenWithClaims(&jwt.StandardClaims{
					Audience:  "nse",
					Subject:   "illegal subject",
				}),
			},
			isMatching: false,
		},
	}

	policyBytes, err := ioutil.ReadFile("../tokensmatching.rego")
	require.Nil(t, err)

	p, err := rego.New(
		rego.Query("data.defaultpolicies.valid_sub_aud_in_path"),
		rego.Module("tokensmatching.rego", string(policyBytes))).PrepareForEval(context.Background())
	require.Nilf(t, err, "failed to create new rego policy: %v", err)

	for i := range suits {
		s := suits[i]

		conn := getConnectionWithTokens(s.tokens)
		require.Nil(t, err)

		input, err := opautils.PreparedOpaInput(conn, nil, opautils.Request, opautils.Client)
		require.Nil(t, err)

		t.Run(s.name, func(t *testing.T) {
			checkResult := func(err error) {
				if s.isMatching {
					require.Nil(t, err)
					return
				}

				require.NotNil(t, err)
				s, ok := status.FromError(err)
				require.True(t, ok, "error without error status code")
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			err = opautils.CheckPolicy(context.Background(), &p, input)
			checkResult(err)
		})
	}
}
