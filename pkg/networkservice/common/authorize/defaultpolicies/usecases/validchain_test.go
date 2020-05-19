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

package usecases

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"github.com/open-policy-agent/opa/rego"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidChain(t *testing.T) {
	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id: "conn",
			Path: &networkservice.Path{
				Index:        0,
				PathSegments: []*networkservice.PathSegment{},
			},
		},
	}

	suits := []struct {
		name            string
		tokenGenerators []token.GeneratorFunc
		isValidChain    bool
	}{
		{
			name:  "simple positive use case with valid chain",
			tokenGenerators: []token.GeneratorFunc{
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "nsc",
					Audience: "nsmgr1",
				}),
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "nsmgr1",
					Audience: "nsmgr2",
				}),
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "nsmgr2",
					Audience: "nse",
				}),
			},
			isValidChain: true,
		},
		{
			name:    "negative use case with broken chain",
			tokenGenerators: []token.GeneratorFunc{
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "nsc",
					Audience: "nsmgr1",
				}),
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "spy subject",
					Audience: "spy audience",
				}),
				TokenGeneratorFunc(&jwt.StandardClaims{
					Subject: "nsmgr2",
					Audience: "nse",
				}),

			},
			isValidChain: false,
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

		elements := []networkservice.NetworkServiceServer{
			adapters.NewClientToServer(updatepath.NewClient("nsc", s.tokenGenerators[0])),
			authorize.NewServer(&p),
			updatepath.NewServer("nsmgr1", s.tokenGenerators[1]),
			authorize.NewServer(&p),
			updatepath.NewServer("nsmgr2", s.tokenGenerators[2]),
			authorize.NewServer(&p),
		}

		t.Run(s.name, func(t *testing.T) {
			checkResult := func(err error) {
				if s.isValidChain {
					require.Nil(t, err)
					return
				}

				require.NotNil(t, err)
				s, ok := status.FromError(err)
				require.True(t, ok, "error without error status code")
				require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
			}

			server := next.NewNetworkServiceServer(elements...)

			_, err := server.Request(context.Background(), request)
			checkResult(err)
		})
	}
}