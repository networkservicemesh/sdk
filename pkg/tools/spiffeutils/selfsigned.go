// Copyright (c) 2020 Cisco and/or its affiliates.
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

// Package spiffeutils provides a variety of utilities for using spiffe to provide tokens, credentials, and grpc options
package spiffeutils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net/url"
	"time"

	"github.com/cloudflare/cfssl/signer"
	"github.com/pkg/errors"
)

// SelfSignedX509SVID - returns  a self signed X509 Spiffe SVID and private Key for the provided spiffeID, notBefore, and notAfter
func SelfSignedX509SVID(spiffeID *url.URL, notBefore, notAfter time.Time) (*x509.Certificate, crypto.PrivateKey, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating SelfSignedX509SVID")
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating SelfSignedX509SVID")
	}
	template, err := X509SVIDTemplate(spiffeID, priv.Public(), notBefore, notAfter, serialNumber)
	if err != nil {
		return nil, nil, err // Intentionally not Wrapping as done in called function
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, priv.Public(), priv)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating SelfSignedX509SVID")
	}
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Error creating SelfSignedX509SVID")
	}
	return cert, priv, nil
}

// X509SVIDTemplate - returns a template for an X509SVID certificate for the provided spiffeID, publicKey, notBefore, notAfter,and serialNumber
func X509SVIDTemplate(spiffeID *url.URL, publicKey crypto.PublicKey, notBefore, notAfter time.Time, serialNumber *big.Int) (*x509.Certificate, error) {
	// Borrowed with love and adapted from https://github.com/spiffe/spire/blob/248127372308a5542a62bb344422d6a2061530b0/pkg/server/ca/templates.go#L42
	// Under that Apache 2.0 License: https://github.com/spiffe/spire/blob/master/LICENSE

	subject := pkix.Name{
		Country:      []string{"US"},
		Organization: []string{"SPIRE"},
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      subject,
		URIs:         []*url.URL{spiffeID},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageKeyAgreement |
			x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
		PublicKey:             publicKey,
	}
	var err error
	template.SubjectKeyId, err = signer.ComputeSKI(template)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating X509SVIDTemplate")
	}

	return template, nil
}
