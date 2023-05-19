// Copyright (c) 2021 Cisco and/or its affiliates.
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

type cancelKey struct{}
type pendingErrorKey struct{}

func setPendingErrorFlag(ctx context.Context, value bool) {
	metadata.Map(ctx, true).Store(pendingErrorKey{}, value)
}

func getPendingErrorFlag(ctx context.Context) bool {
	rawValue, ok := metadata.Map(ctx, true).Load(pendingErrorKey{})
	if !ok {
		return false
	}
	value, ok := rawValue.(bool)
	if !ok {
		return false
	}
	return value
}

// storeCancel sets the context.CancelFunc stored in per Connection.Id metadata.
func storeCancel(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(cancelKey{}, cancel)
}

// loadAndDelete deletes the context.CancelFunc stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDelete(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(cancelKey{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}
