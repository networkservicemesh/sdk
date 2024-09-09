// Copyright (c) 2022 Cisco and/or its affiliates.
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

package authorize_test

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	spiffeid1 = "spiffe://test.com/workload1"
	spiffeid2 = "spiffe://test.com/workload2"
)

func genJWTWithClaims(claims *jwt.RegisteredClaims) string {
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("super secret"))
	return t
}

func genTokenFunc(claims *jwt.RegisteredClaims) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (string, time.Time, error) {
		return genJWTWithClaims(claims), time.Time{}, nil
	}
}

func getPath(t *testing.T, spiffeID string) *grpcmetadata.Path {
	segments := []struct {
		name           string
		tokenGenerator token.GeneratorFunc
	}{
		{
			name: spiffeID,
			tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
				Subject:  spiffeID,
				Audience: []string{"nsmgr"},
			}),
		},
		{
			name: "nsmgr",
			tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
				Subject:  "nsmgr",
				Audience: []string{"forwarder"},
			}),
		},
		{
			name: "forwarder",
			tokenGenerator: genTokenFunc(&jwt.RegisteredClaims{
				Subject:  "forwarder",
				Audience: []string{"nse"},
			}),
		},
	}

	path := &grpcmetadata.Path{
		PathSegments: []*grpcmetadata.PathSegment{},
	}

	for _, segment := range segments {
		tok, _, err := segment.tokenGenerator(nil)
		require.NoError(t, err)
		path.PathSegments = append(path.PathSegments, &grpcmetadata.PathSegment{Token: tok})
	}

	return path
}
