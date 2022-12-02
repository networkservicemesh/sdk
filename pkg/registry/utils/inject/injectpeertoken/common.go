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

package injectpeertoken

import (
	"context"
	"time"

	"google.golang.org/grpc/metadata"
)

const (
	tokenKey      = "nsm-client-token"
	expireTimeKey = "nsm-client-token-expires"
)

func withPeerToken(ctx context.Context, peerToken string) context.Context {
	m, _ := metadata.FromIncomingContext(ctx)
	if m == nil {
		m = metadata.New(make(map[string]string))
	}
	m[tokenKey] = []string{peerToken}
	m[expireTimeKey] = []string{time.Now().Add(time.Hour).Format(time.RFC3339Nano)}
	ctx = metadata.NewIncomingContext(ctx, m)
	return ctx
}
