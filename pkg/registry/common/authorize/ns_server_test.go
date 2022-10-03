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
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"go.uber.org/goleak"
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

func TestAuthzNetworkServiceRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	server := authorize.NewNetworkServiceRegistryServer()

	u1, _ := url.Parse("spiffe://test.com/workload1")
	certBytes := generateCert(u1)
	x509cert, _ := x509.ParseCertificate(certBytes)

	authinfo := credentials.TLSInfo{
		State: tls.ConnectionState{
			PeerCertificates: []*x509.Certificate{
				x509cert,
			},
		},
	}

	ctx := context.Background()
	ctx, _ = withPeer(ctx, certBytes)

	tokenFunc1, expires1, err := genTokenFunc(&jwt.RegisteredClaims{
		Subject:   "nsc",
		Audience:  []string{"nsmgr"},
		ExpiresAt: jwt.NewNumericDate(time.Date(2023, 1, 1, 1, 1, 1, 1, time.UTC)),
	})(authinfo)

	tokenFunc2, expires2, err := genTokenFunc(&jwt.RegisteredClaims{
		Subject:   "nsmgr",
		Audience:  []string{"registry"},
		ExpiresAt: jwt.NewNumericDate(time.Date(2023, 1, 1, 1, 1, 1, 1, time.UTC)),
	})(authinfo)

	tokenFunc3, expires3, err := genTokenFunc(&jwt.RegisteredClaims{
		Subject:   "registry",
		Audience:  []string{"registry"},
		ExpiresAt: jwt.NewNumericDate(time.Date(2023, 1, 1, 1, 1, 1, 1, time.UTC)),
	})(authinfo)

	nsReg := &registry.NetworkService{
		Name: "ns-1",
		Path: &registry.Path{
			Index: 2,
			PathSegments: []*registry.PathSegment{
				{
					Name:    "nse",
					Token:   tokenFunc1,
					Expires: timestamppb.New(expires1),
				},
				{
					Name:    "nsmgr",
					Token:   tokenFunc2,
					Expires: timestamppb.New(expires2),
				},
				{
					Name:    "registry",
					Token:   tokenFunc3,
					Expires: timestamppb.New(expires3),
				},
			},
		},
	}

	_, err = server.Register(ctx, nsReg)
	require.NoError(t, err)

}
