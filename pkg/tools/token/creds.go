// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package token

import (
	"context"
	"errors"
	"time"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
)

type perRPCCredentials struct {
	genTokenFunc GeneratorFunc
}

func (creds *perRPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errors.New("peer is missed in ctx")
	}

	var authInfo = credentials.AuthInfo(nil)

	if p != nil {
		authInfo = p.AuthInfo
	}

	tok, expire, err := creds.genTokenFunc(authInfo)

	if err != nil {
		return nil, err
	}

	return map[string]string{
		tokenKey:      tok,
		expireTimeKey: expire.Format(time.RFC3339Nano),
	}, nil
}

func (creds *perRPCCredentials) RequireTransportSecurity() bool {
	return true
}

var _ credentials.PerRPCCredentials = (*perRPCCredentials)(nil)

// NewPerRPCCredentials creates credentials.PerRPCCredentials based on token generator func
func NewPerRPCCredentials(genTokenFunc GeneratorFunc) credentials.PerRPCCredentials {
	if genTokenFunc == nil {
		panic("genTokenFunc cannot be nil")
	}

	return &perRPCCredentials{
		genTokenFunc: genTokenFunc,
	}
}
