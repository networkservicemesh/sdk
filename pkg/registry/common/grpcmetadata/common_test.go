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

package grpcmetadata_test

import (
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"

	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/sdk/pkg/tools/token"
)

const (
	key = "supersecret"
)

// tokenGeneratorFunc generates new tokens automatically (based on time change).
// time.Second + smth - the time tick for jwt is a second.
func tokenGeneratorFunc(clock *clockmock.Mock, spiffeID string) token.GeneratorFunc {
	return func(peerAuthInfo credentials.AuthInfo) (string, time.Time, error) {
		clock.Add(time.Second + time.Millisecond*10)
		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": spiffeID,
			"exp": jwt.NewNumericDate(clock.Now().Add(time.Hour)),
		},
		).SignedString([]byte(key))
		return tok, clock.Now(), err
	}
}
