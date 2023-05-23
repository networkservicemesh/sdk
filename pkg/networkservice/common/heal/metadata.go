// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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

package heal

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type cancelFullKey struct{}
type cancelDataKey struct{}

// storeCancelFull saves CancelFunc for the full (control plane + data plane) monitoring
func storeCancelFull(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(cancelFullKey{}, cancel)
}

// loadAndDeleteFull deletes CancelFunc for the full (control plane + data plane) monitoring,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDeleteFull(ctx context.Context) (value context.CancelFunc, loaded bool) {
	rawValue, loaded := metadata.Map(ctx, true).LoadAndDelete(cancelFullKey{})
	if !loaded {
		return
	}
	value, loaded = rawValue.(context.CancelFunc)
	return value, loaded
}

// storeCancelFull saves CancelFunc for the data-plane-only monitoring
func storeCancelData(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(cancelDataKey{}, cancel)
}

// loadAndDeleteFull deletes CancelFunc for the data-plane-only monitoring,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDeleteData(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(cancelDataKey{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}
