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
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc/metadata"
)

// #nosec
const tokenKey = "nsm-client-token"

// #nosec
const expireTimeKey = "nsm-client-token-expires"

// FromContext returns token from the incoming context metadata.
func FromContext(ctx context.Context) (string, time.Time, error) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return "", time.Time{}, errors.New("metadata is missed in ctx")
	}

	token, err := getSingleValue(md, tokenKey)

	if err != nil {
		return "", time.Time{}, err
	}

	rawExpiration, err := getSingleValue(md, expireTimeKey)

	if err != nil {
		return "", time.Time{}, err
	}

	expireTime, err := time.Parse(time.RFC3339Nano, rawExpiration)

	if err != nil {
		return "", time.Time{}, errors.WithStack(err)
	}

	return token, expireTime, nil
}

func getSingleValue(md metadata.MD, key string) (string, error) {
	values := md.Get(key)

	if len(values) != 1 {
		return "", errors.Errorf("values of '%v' has wrong length %v, expected 1", key, len(values))
	}

	return values[0], nil
}
