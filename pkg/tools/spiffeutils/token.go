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

package spiffeutils

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/uri"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// SpiffeJWTTokenGeneratorFunc - creates a token.GeneratorFunc that creates spiffe JWT tokens from the cert returned by getCert()
func SpiffeJWTTokenGeneratorFunc(getCert func() (*tls.Certificate, error), maxTokenLifeTime time.Duration) token.GeneratorFunc {
	return func(authInfo credentials.AuthInfo) (string, time.Time, error) {
		ownCert, err := getCert()
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "Error creating Token")
		}
		x509ownCert, err := X509Cert(ownCert)
		if err != nil {
			return "", time.Time{}, err
		}

		expireTime := time.Now().Add(maxTokenLifeTime)
		if x509ownCert.NotAfter.Before(expireTime) {
			expireTime = ownCert.Leaf.NotAfter
		}
		ownSpiffeID, err := SpiffeIDFromTLS(ownCert)
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "Error creating Token")
		}
		claims := jwt.StandardClaims{
			Subject:   ownSpiffeID,
			ExpiresAt: expireTime.Unix(),
		}
		if authInfo != nil {
			if tlsInfo, ok := authInfo.(credentials.TLSInfo); ok {
				if len(tlsInfo.State.PeerCertificates) > 0 {
					peerCert := tlsInfo.State.PeerCertificates[0]
					peerSpiffeID, err2 := SpiffeIDFromX509(peerCert)
					if err2 != nil {
						return "", time.Time{}, err2
					}
					if peerCert.NotAfter.Before(expireTime) {
						expireTime = peerCert.NotAfter
					}
					claims.Audience = peerSpiffeID
				}
			}
		}
		tok, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(ownCert.PrivateKey)
		return tok, expireTime, err
	}
}

// X509Cert - convert *tls.Certificate to *x509.Certificate
func X509Cert(cert *tls.Certificate) (*x509.Certificate, error) {
	if cert == nil || cert.Certificate == nil {
		return nil, errors.Errorf("Error converting TLS to X509 Cert: cert must not be nil: %+v", nil)
	}
	if cert.Leaf == nil {
		leaf, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			return nil, errors.Errorf("Error converting TLS to X509 Cert: TLS Cert Missing x509 certificate: %+v", err)
		}
		cert.Leaf = leaf
	}
	return cert.Leaf, nil
}

// SpiffeIDFromX509 extracts spiffeID from *x509.Certificate
func SpiffeIDFromX509(cert *x509.Certificate) (string, error) {
	spiffeIDs, err := uri.GetURINamesFromCertificate(cert)
	if err != nil {
		return "", errors.Wrap(err, "Error extracting SpiffeID from cert")
	}
	if len(spiffeIDs) < 1 {
		return "", errors.New("Error extracting SpiffeID from cert: No SpiffeIDs for self in TLS tlsCert")
	}
	return spiffeIDs[0], nil
}

// SpiffeIDFromTLS extracts spiffeID from *tls.Certificate
func SpiffeIDFromTLS(cert *tls.Certificate) (string, error) {
	x509cert, err := X509Cert(cert)
	if err != nil {
		return "", err
	}
	spiffeID, err := SpiffeIDFromX509(x509cert)
	if err != nil {
		return "", err
	}
	return spiffeID, nil
}
