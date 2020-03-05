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
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authn"
	"github.com/networkservicemesh/sdk/pkg/tools/security"
	securitytest "github.com/networkservicemesh/sdk/pkg/tools/security/test"
)

type AuthnServerTestSuite struct {
	suite.Suite
	TestCA             tls.Certificate
	TestTLSCertificate tls.Certificate
	AuthnServer        networkservice.NetworkServiceServer
}

func (s *AuthnServerTestSuite) SetupSuite() {
	var err error
	s.TestCA, err = securitytest.GenerateCA()
	if err != nil {
		panic(err)
	}

	s.TestTLSCertificate, err = securitytest.GenerateKeyPair("spiffe://test.com/workload", "test.com", &s.TestCA)
	if err != nil {
		panic(err)
	}
}

func TestAuthnServerTestSuite(t *testing.T) {
	suite.Run(t, new(AuthnServerTestSuite))
}

func (s *AuthnServerTestSuite) TestAuthnServer_Request() {
	p := &securitytest.Provider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &s.TestTLSCertificate, nil
		},
	}

	srv := authn.NewServer(p)

	x509crt, err := x509.ParseCertificate(s.TestTLSCertificate.Certificate[0])
	s.Nil(err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					x509crt,
				},
			},
		},
	})

	token, err := security.GenerateToken(context.Background(), p, 0)
	s.Nil(err)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
			Path: &networkservice.Path{
				Index: 0,
				PathSegments: []*networkservice.PathSegment{
					{
						Id:    "conn-1",
						Token: token,
					},
				},
			},
		},
	}

	_, err = srv.Request(ctx, request)
	s.Nil(err)
}

func (s *AuthnServerTestSuite) TestAuthnServer_UnauthorizedRequest() {
	p := &securitytest.Provider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &s.TestTLSCertificate, nil
		},
	}

	srv := authn.NewServer(p)

	wrongX509crt, err := x509.ParseCertificate(s.TestCA.Certificate[0])
	s.Nil(err)

	ctx := peer.NewContext(context.Background(), &peer.Peer{
		AuthInfo: credentials.TLSInfo{
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{
					wrongX509crt,
				},
			},
		},
	})

	token, err := security.GenerateToken(context.Background(), p, 0)
	s.Nil(err)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
			Path: &networkservice.Path{
				Index: 0,
				PathSegments: []*networkservice.PathSegment{
					{
						Id:    "conn-1",
						Token: token,
					},
				},
			},
		},
	}

	_, err = srv.Request(ctx, request)
	s.Equal(status.Error(codes.Unauthenticated, "unable to verify token: crypto/ecdsa: verification error"), err)
}
