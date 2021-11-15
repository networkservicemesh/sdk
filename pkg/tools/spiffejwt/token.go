// Copyright (c) 2020-2021 Cisco and/or its affiliates.
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

package spiffejwt

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

// TokenGeneratorFunc - creates a token.TokenGeneratorFunc that creates spiffe JWT tokens from the cert returned by getCert()
func TokenGeneratorFunc(source x509svid.Source, maxTokenLifeTime time.Duration) token.GeneratorFunc {
	return func(authInfo credentials.AuthInfo) (string, time.Time, error) {
		ownSVID, err := source.GetX509SVID()
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "Error creating Token")
		}

		expireTime := time.Now().Add(maxTokenLifeTime)
		if ownSVID.Certificates[0].NotAfter.Before(expireTime) {
			expireTime = ownSVID.Certificates[0].NotAfter
		}
		if err != nil {
			return "", time.Time{}, errors.Wrap(err, "Error creating Token")
		}
		claims := jwt.RegisteredClaims{
			Subject:   ownSVID.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expireTime),
		}
		if authInfo != nil {
			if tlsInfo, ok := authInfo.(credentials.TLSInfo); ok {
				if len(tlsInfo.State.PeerCertificates) > 0 {
					peerCert := tlsInfo.State.PeerCertificates[0]
					peerSpiffeID, err2 := x509svid.IDFromCert(peerCert)
					if err2 != nil {
						return "", time.Time{}, err2
					}
					if peerCert.NotAfter.Before(expireTime) {
						expireTime = peerCert.NotAfter
					}
					claims.Audience = []string{peerSpiffeID.String()}
				}
			}
		}
		tok, err := jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(ownSVID.PrivateKey)
		return tok, expireTime, err
	}
}
