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

package opa_test

import (
	"context"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"

	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

type serviceSample struct {
	tokenGenerator token.GeneratorFunc
	name           string
}

type chainSample struct {
	name         string
	server       []serviceSample
	isValidChain bool
}

func getSamples() []chainSample {
	return []chainSample{
		{
			name: "Valid chain",
			server: []serviceSample{
				{
					name: "nsc",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "nsc",
						Audience: "nsmgr1",
					}),
				},
				{
					name: "nsmgr1",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "nsmgr1",
						Audience: "nsmgr2",
					}),
				},
				{
					name: "nsmgr2",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "nsmgr2",
						Audience: "nse",
					}),
				},
			},
			isValidChain: true,
		},
		{
			name: "Invalid chain",
			server: []serviceSample{
				{
					name: "nsc",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "nsc",
						Audience: "nsmgr1",
					}),
				},
				{
					name: "nsmgr1",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "nsmgr1",
						Audience: "nsmgr2",
					}),
				},
				{
					name: "nsmgr2",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject:  "spy",
						Audience: "nse",
					}),
				},
				{
					name: "nse",
					tokenGenerator: genTokenFunc(&jwt.StandardClaims{
						Subject: "nse",
					}),
				},
			},
			isValidChain: false,
		},
	}
}

func TestWithTokensPathValidPolicy(t *testing.T) {
	genRequest := func() *networkservice.NetworkServiceRequest {
		return &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id: "conn",
				Path: &networkservice.Path{
					Index:        0,
					PathSegments: []*networkservice.PathSegment{},
				},
			},
		}
	}

	p := opa.WithTokenChainPolicy()
	samples := getSamples()

	for i := range samples {
		s := samples[i]
		checkResult := func(err error) {
			if s.isValidChain {
				require.Nil(t, err)
				return
			}

			require.NotNil(t, err)
			s, ok := status.FromError(errors.Cause(err))
			require.True(t, ok, "error without error status code", err.Error())
			require.Equal(t, s.Code(), codes.PermissionDenied, "wrong error status code")
		}
		var servers []networkservice.NetworkServiceServer
		for _, server := range s.server {
			servers = append(servers,
				chain.NewNetworkServiceServer(
					updatepath.NewServer(server.name),
					updatetoken.NewServer(server.tokenGenerator),
					authorize.NewServer(authorize.WithPolicies(p))),
			)
		}

		t.Run(s.name, func(t *testing.T) {
			var err error
			var request = genRequest()
			for i := 0; i < len(servers); i++ {
				request.Connection, err = servers[i].Request(context.Background(), request)
				if err != nil {
					break
				}
			}
			checkResult(err)
		})
	}
}
