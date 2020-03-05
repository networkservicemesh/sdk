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

package authn_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authn"
	"github.com/networkservicemesh/sdk/pkg/tools/security"
	securitytest "github.com/networkservicemesh/sdk/pkg/tools/security/test"
)

func TestAuthnClient_Request(t *testing.T) {
	testCA, err := securitytest.GenerateCA()
	require.Nil(t, err)

	testTLSCertificate, err := securitytest.GenerateKeyPair("spiffe://test.com/workload", "test.com", &testCA)
	require.Nil(t, err)

	p := &securitytest.Provider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &testTLSCertificate, nil
		},
	}

	client := authn.NewClient(p)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
			Path: &networkservice.Path{
				Index: 0,
				PathSegments: []*networkservice.PathSegment{
					{
						Id: "conn-1",
					},
				},
			},
		},
	}

	conn, err := client.Request(context.Background(), request)
	require.Nil(t, err)
	require.NotEmpty(t, conn.GetPath().GetPathSegments()[0].Token)

	x509crt, err := x509.ParseCertificate(testTLSCertificate.Certificate[0])
	require.Nil(t, err)
	err = security.VerifyToken(conn.GetPath().GetPathSegments()[0].Token, x509crt)
	require.Nil(t, err)
}
