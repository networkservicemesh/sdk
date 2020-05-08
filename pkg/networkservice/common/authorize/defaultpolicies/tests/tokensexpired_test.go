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
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/opautils"

	"github.com/dgrijalva/jwt-go"
	"github.com/open-policy-agent/opa/rego"
	"github.com/stretchr/testify/require"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)


func TestNoTokensExpiredPolicy(t *testing.T){
	suits := []struct {
		name     string
		tokens  []string
		isNotExpired bool
	}{
		{
			name:  "simple positive test with one token",
			tokens: []string{
				generateTokenWithClaims(&jwt.StandardClaims{
					ExpiresAt: time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
				}),
			},
			isNotExpired: true,
		},
		{
			name:    "negative test with expired/not expired tokens",
			tokens: []string{
				generateTokenWithClaims(&jwt.StandardClaims{
					ExpiresAt: time.Date(3000, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
				}),
				generateTokenWithClaims(&jwt.StandardClaims{
					ExpiresAt: time.Date(2000, 1, 1, 1, 1, 1, 1, time.UTC).Unix(),
				}),
			},
			isNotExpired: false,
		},
	}

	policyBytes, err := ioutil.ReadFile("../tokensexpired.rego")
	require.Nil(t, err)

	p, err := rego.New(
		rego.Query("data.defaultpolicies.no_tokens_expired"),
		rego.Module("tokensexpired.rego", string(policyBytes))).PrepareForEval(context.Background())
	require.Nilf(t, err, "failed to create new rego policy: %v", err)

	for i := range suits {
		s := suits[i]

		conn := getConnectionWithTokens(s.tokens)
		require.Nil(t, err)

		input, err := opautils.PreparedOpaInput(conn, nil, opautils.Request, opautils.Client)
		require.Nil(t, err)

		t.Run(s.name, func(t *testing.T) {
			checkResult := func(err error) {
				if s.isNotExpired {
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