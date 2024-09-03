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

package upstreamrefresh

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type (
	key         struct{}
	keyNotifier struct{}
)

// store sets the context.CancelFunc stored in per Connection.Id metadata.
func store(ctx context.Context, cancel context.CancelFunc) {
	metadata.Map(ctx, true).Store(key{}, cancel)
}

// loadAndDelete deletes the context.CancelFunc stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func loadAndDelete(ctx context.Context) (value context.CancelFunc, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(context.CancelFunc)
	return value, ok
}

// storeLocalNotifier sets the Notifier stored in per Connection.Id metadata.
func storeLocalNotifier(ctx context.Context, isClient bool, notifier Notifier) {
	metadata.Map(ctx, isClient).Store(keyNotifier{}, notifier)
}

// LoadLocalNotifier loads Notifier stored in per Connection.Id metadata.
// The loaded result reports whether the key was present.
func LoadLocalNotifier(ctx context.Context, isClient bool) (value Notifier, ok bool) {
	rawValue, ok := metadata.Map(ctx, isClient).Load(keyNotifier{})
	if !ok {
		return
	}
	value, ok = rawValue.(Notifier)
	return value, ok
}
