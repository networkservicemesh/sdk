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

package security_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	securitytest "github.com/networkservicemesh/sdk/pkg/tools/security/test"

	"github.com/dgrijalva/jwt-go"

	"github.com/networkservicemesh/sdk/pkg/tools/security"
)

const (
	spiffeID = "spiffe://test.com/workload"
)

type TokenTestSuite struct {
	suite.Suite
	TestCA             tls.Certificate
	TestTLSCertificate tls.Certificate
}

func (s *TokenTestSuite) SetupSuite() {
	var err error
	s.TestCA, err = securitytest.GenerateCA()
	if err != nil {
		panic(err)
	}

	s.TestTLSCertificate, err = securitytest.GenerateKeyPair(spiffeID, "test.com", &s.TestCA)
	if err != nil {
		panic(err)
	}
}

func TestTokenTestSuite(t *testing.T) {
	suite.Run(t, new(TokenTestSuite))
}

type testProvider struct {
	GetCertificateFunc func(ctx context.Context) (*tls.Certificate, error)
}

func (t *testProvider) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	panic("implement me")
}

func (t *testProvider) GetCertificate(ctx context.Context) (*tls.Certificate, error) {
	return t.GetCertificateFunc(ctx)
}

func (s *TokenTestSuite) TestGenerateToken() {
	p := &testProvider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &s.TestTLSCertificate, nil
		},
	}

	token, err := security.GenerateToken(context.Background(), p, 0)
	s.Nil(err)

	x509crt, err := x509.ParseCertificate(s.TestTLSCertificate.Certificate[0])
	s.Nil(err)

	_, err = new(jwt.Parser).Parse(token, func(token *jwt.Token) (interface{}, error) {
		return x509crt.PublicKey, nil
	})
	s.Nil(err)
}

func (s *TokenTestSuite) TestGenerateToken_Expire() {
	p := &testProvider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &s.TestTLSCertificate, nil
		},
	}

	token, err := security.GenerateToken(context.Background(), p, 3*time.Second)
	s.Nil(err)

	<-time.After(5 * time.Second)

	x509crt, err := x509.ParseCertificate(s.TestTLSCertificate.Certificate[0])
	s.Nil(err)

	_, err = new(jwt.Parser).Parse(token, func(token *jwt.Token) (interface{}, error) {
		return x509crt.PublicKey, nil
	})
	s.NotNil(err)
}

func (s *TokenTestSuite) TestVerifyToken() {
	token, err := jwt.New(jwt.SigningMethodES256).SignedString(s.TestTLSCertificate.PrivateKey)
	s.Nil(err)

	x509crt, err := x509.ParseCertificate(s.TestTLSCertificate.Certificate[0])
	s.Nil(err)

	err = security.VerifyToken(token, x509crt)
	s.Nil(err)

	invalidX509crt, err := x509.ParseCertificate(s.TestCA.Certificate[0])
	s.Nil(err)

	err = security.VerifyToken(token, invalidX509crt)
	s.NotNil(err)
}
