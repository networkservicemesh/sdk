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

// Package clientconn allows storing grpc.ClientConnInterface stored in per Connection.Id metadata
package clientconn

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
)

type key struct{}

// Store sets the grpc.ClientConnInterface stored in per Connection.Id metadata.
func Store(ctx context.Context, cc grpc.ClientConnInterface) {
	metadata.Map(ctx, true).Store(key{}, cc)
}

// Delete deletes the grpc.ClientConnInterface stored in per Connection.Id metadata
func Delete(ctx context.Context) {
	metadata.Map(ctx, true).Delete(key{})
}

// Load returns the grpc.ClientConnInterface stored in per Connection.Id metadata, or nil if no
// value is present.
// The ok result indicates whether value was found in the per Connection.Id metadata.
func Load(ctx context.Context) (value grpc.ClientConnInterface, ok bool) {
	m := metadata.Map(ctx, true)
	rawValue, ok := m.Load(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(grpc.ClientConnInterface)
	return value, ok
}

// LoadOrStore returns the existing grpc.ClientConnInterface stored in per Connection.Id metadata if present.
// Otherwise, it stores and returns the given grpc.ClientConnInterface.
// The loaded result is true if the value was loaded, false if stored.
func LoadOrStore(ctx context.Context, cc grpc.ClientConnInterface) (value grpc.ClientConnInterface, ok bool) {
	rawValue, ok := metadata.Map(ctx, true).LoadOrStore(key{}, cc)
	if !ok {
		return cc, ok
	}
	value, ok = rawValue.(grpc.ClientConnInterface)
	return value, ok
}

// LoadAndDelete deletes the grpc.ClientConnInterface stored in per Connection.Id metadata,
// returning the previous value if any. The loaded result reports whether the key was present.
func LoadAndDelete(ctx context.Context) (value grpc.ClientConnInterface, ok bool) {
	m := metadata.Map(ctx, true)
	rawValue, ok := m.LoadAndDelete(key{})
	if !ok {
		return
	}
	value, ok = rawValue.(grpc.ClientConnInterface)
	return value, ok
}
