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

package cleanup

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type (
	keyCancel struct{}
	keyCC     struct{}
)

// storeCancel sets the context.CancelFunc stored in per Connection.Id metadata.
func storeCancel(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(keyCancel{}, cancel)
}

// loadAndDeleteCancel deletes the context.CancelFunc stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDeleteCancel(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(keyCancel{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}

// storeCC sets the flag to delete clientconn in per Connection.Id metadata.
func storeCC(ctx context.Context) {
	metadata.Map(ctx, true).Store(keyCC{}, struct{}{})
}

// loadAndDeleteCC deletes the flag stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDeleteCC(ctx context.Context) (value struct{}, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(keyCC{})
	if !ok {
		return
	}
	value, ok = rawValue.(struct{})
	return value, ok
}
