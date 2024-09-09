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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	nameTypeURI = 6
)

func generateCA() (tls.Certificate, error) {
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		BasicConstraintsValid: true,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to ecdsa.GenerateKey")
	}

	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to x509.CreateCertificate")
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to x509.MarshalECPrivateKey")
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}

func marshalSAN(spiffeID string) ([]byte, error) {
	return asn1.Marshal([]asn1.RawValue{{Tag: nameTypeURI, Class: 2, Bytes: []byte(spiffeID)}})
}

func generateKeyPair(spiffeID, domain string, caTLS *tls.Certificate) (tls.Certificate, error) {
	san, err := marshalSAN(spiffeID)
	if err != nil {
		return tls.Certificate{}, nil
	}

	oidSanExtension := []int{2, 5, 29, 17}
	cert := &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Organization:  []string{"ORGANIZATION_NAME"},
			Country:       []string{"COUNTRY_CODE"},
			Province:      []string{"PROVINCE"},
			Locality:      []string{"CITY"},
			StreetAddress: []string{"ADDRESS"},
			PostalCode:    []string{"POSTAL_CODE"},
			CommonName:    domain,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(10, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		ExtraExtensions: []pkix.Extension{
			{
				Id:    oidSanExtension,
				Value: san,
			},
		},
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		KeyUsage:           x509.KeyUsageDigitalSignature,
	}

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil
	}

	pub := &priv.PublicKey

	ca, err := x509.ParseCertificate(caTLS.Certificate[0])
	if err != nil {
		return tls.Certificate{}, nil
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, pub, caTLS.PrivateKey)
	if err != nil {
		return tls.Certificate{}, nil
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, errors.Wrap(err, "failed to x509.MarshalECPrivateKey")
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}

func genJWTWithClaimsWithYear(year int) string {
	return genJWTWithClaims(&jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Date(year, 1, 1, 1, 1, 1, 1, time.UTC)),
	})
}

func genJWTWithClaims(claims *jwt.RegisteredClaims) string {
	t, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("super secret"))
	return t
}

func genConnectionWithTokens(tokens []string) *networkservice.Connection {
	rv := &networkservice.Connection{
		Path: &networkservice.Path{
			PathSegments: []*networkservice.PathSegment{},
		},
	}

	for _, token := range tokens {
		rv.Path.PathSegments = append(rv.Path.PathSegments, &networkservice.PathSegment{
			Token: token,
		})
	}

	return rv
}

func genTokenFunc(claims *jwt.RegisteredClaims) token.GeneratorFunc {
	return func(_ credentials.AuthInfo) (string, time.Time, error) {
		return genJWTWithClaims(claims), time.Time{}, nil
	}
}
