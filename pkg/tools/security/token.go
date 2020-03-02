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

package security

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// GenerateToken generates JWT token based on tls.Certificate from security.Provider
func GenerateToken(ctx context.Context, p Provider, expiresAt time.Duration) (string, error) {
	crt, err := p.GetCertificate(ctx)
	if err != nil {
		return "", err
	}

	claims := jwt.StandardClaims{
		ExpiresAt: time.Now().Add(expiresAt).Unix(),
	}

	return jwt.NewWithClaims(jwt.SigningMethodES256, claims).SignedString(crt.PrivateKey)
}

// VerifyToken verifies JWT 'token' using x509.Certificate
func VerifyToken(token string, x509crt *x509.Certificate) error {
	_, err := new(jwt.Parser).Parse(token, func(token *jwt.Token) (interface{}, error) {
		return x509crt.PublicKey, nil
	})
	return err
}
