// Copyright (c) 2020-2022 Cisco and/or its affiliates.
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

// Package token provides a simple type for functions that generate tokens
package token

import (
	"time"

	"google.golang.org/grpc/credentials"
)

// GeneratorFunc - a function which takes the credentials.AuthInfo of the peer of the client or server
//
//	and returns a token as a string (example: JWT), a expireTime, and an error.
type GeneratorFunc func(peerAuthInfo credentials.AuthInfo) (token string, expireTime time.Time, err error)
