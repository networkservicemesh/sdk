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

package clientconn

import (
	"context"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
)

type mapKey struct{}
type nameKey struct{}

func withClientConnMetadata(ctx context.Context, m *stringCCMap, key string) context.Context {
	ctx = context.WithValue(ctx, nameKey{}, key)
	ctx = context.WithValue(ctx, mapKey{}, m)
	return ctx
}

func nameFromContext(ctx context.Context) string {
	if v := ctx.Value(nameKey{}); v != nil {
		return v.(string)
	}
	if u := clienturlctx.ClientURL(ctx); u != nil {
		return u.String()
	}
	return ""
}

// LoadAndDelete -
func LoadAndDelete(ctx context.Context) (grpc.ClientConnInterface, bool) {
	k := nameFromContext(ctx)

	if v, ok := ctx.Value(mapKey{}).(*stringCCMap); ok && k != "" {
		return v.LoadAndDelete(k)
	}

	return nil, false
}

// Store -
func Store(ctx context.Context, cc grpc.ClientConnInterface) {
	k := nameFromContext(ctx)

	if v, ok := ctx.Value(mapKey{}).(*stringCCMap); ok && k != "" {
		v.Store(k, cc)
	}
}

// Delete -
func Delete(ctx context.Context) {
	k := nameFromContext(ctx)

	if v, ok := ctx.Value(mapKey{}).(*stringCCMap); ok && k != "" {
		v.Delete(k)
	}
}

// Load -
func Load(ctx context.Context) (grpc.ClientConnInterface, bool) {
	k := nameFromContext(ctx)

	if v, ok := ctx.Value(mapKey{}).(*stringCCMap); ok && k != "" {
		return v.Load(k)
	}

	return nil, false
}

// LoadOrStore -
func LoadOrStore(ctx context.Context, cc grpc.ClientConnInterface) (grpc.ClientConnInterface, bool) {
	k := nameFromContext(ctx)

	if v, ok := ctx.Value(mapKey{}).(*stringCCMap); ok && k != "" {
		return v.LoadOrStore(k, cc)
	}

	return cc, false
}
