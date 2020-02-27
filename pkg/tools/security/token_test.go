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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"github.com/dgrijalva/jwt-go"
	"github.com/networkservicemesh/sdk/pkg/tools/security"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

const (
	spiffeID    = "spiffe://test.com/workload"
	nameTypeURI = 6
)

var (
	testCA             tls.Certificate
	testTLSCertificate tls.Certificate
)

func init() {
	var err error
	testCA, err = generateCA()
	if err != nil {
		panic(err)
	}

	testTLSCertificate, err = generateKeyPair(spiffeID, "test.com", &testCA)
	if err != nil {
		panic(err)
	}
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
		return tls.Certificate{}, err
	}
	pub := &priv.PublicKey

	certBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, pub, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
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
		return tls.Certificate{}, err
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	return tls.X509KeyPair(certPem, keyPem)
}

func TestGenerateToken(t *testing.T) {
	p := &testProvider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &testTLSCertificate, nil
		},
	}

	token, err := security.GenerateToken(context.Background(), p, 0)
	require.Nil(t, err)

	_, err = new(jwt.Parser).Parse(token, func(token *jwt.Token) (interface{}, error) {
		x509crt, err := x509.ParseCertificate(testTLSCertificate.Certificate[0])
		require.Nil(t, err)
		return x509crt.PublicKey, nil
	})
	require.Nil(t, err)
}

func TestGenerateToken_Expire(t *testing.T) {
	p := &testProvider{
		GetCertificateFunc: func(ctx context.Context) (certificate *tls.Certificate, e error) {
			return &testTLSCertificate, nil
		},
	}

	token, err := security.GenerateToken(context.Background(), p, 3*time.Second)
	require.Nil(t, err)

	<-time.After(5 * time.Second)

	_, err = new(jwt.Parser).Parse(token, func(token *jwt.Token) (interface{}, error) {
		x509crt, err := x509.ParseCertificate(testTLSCertificate.Certificate[0])
		require.Nil(t, err)
		return x509crt.PublicKey, nil
	})
	require.NotNil(t, err)
}

func TestVerifyToken(t *testing.T) {
	token, err := jwt.New(jwt.SigningMethodES256).SignedString(testTLSCertificate.PrivateKey)
	require.Nil(t, err)

	x509crt, err := x509.ParseCertificate(testTLSCertificate.Certificate[0])
	require.Nil(t, err)

	err = security.VerifyToken(token, x509crt)
	require.Nil(t, err)

	invalidX509crt, err := x509.ParseCertificate(testCA.Certificate[0])
	require.Nil(t, err)

	err = security.VerifyToken(token, invalidX509crt)
	require.NotNil(t, err)
}
