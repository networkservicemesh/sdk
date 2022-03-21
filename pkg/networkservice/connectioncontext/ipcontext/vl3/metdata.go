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

package vl3

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type addressKey struct{}

func loadAddress(ctx context.Context) (string, bool) {
	v, ok := metadata.Map(ctx, false).Load(addressKey{})
	if ok {
		return v.(string), true
	}

	return "", false
}

func storeAddress(ctx context.Context, address string) {
	metadata.Map(ctx, false).Store(addressKey{}, address)
}

type cancelKey struct{}

func storeCancel(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(cancelKey{}, cancel)
}

func loadAndDeleteCancel(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(cancelKey{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}
