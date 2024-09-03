// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package externalips

import (
	"context"
	"net"
)

type contextKey string

const (
	internalToExternalKey contextKey = "internal-external"
	externalToInternalKey contextKey = "external-internal"
)

func withInternalReplacer(ctx context.Context, replacer func(net.IP) net.IP) context.Context {
	return context.WithValue(ctx, internalToExternalKey, replacer)
}

func withExternalReplacer(ctx context.Context, replacer func(net.IP) net.IP) context.Context {
	return context.WithValue(ctx, externalToInternalKey, replacer)
}

// ToInternal resolves a external IP to internal.
func ToInternal(ctx context.Context, ip net.IP) net.IP {
	if v := ctx.Value(externalToInternalKey); v != nil {
		return v.(func(net.IP) net.IP)(ip)
	}
	return nil
}

// FromInternal resolves a internal IP to external.
func FromInternal(ctx context.Context, ip net.IP) net.IP {
	if v := ctx.Value(internalToExternalKey); v != nil {
		return v.(func(net.IP) net.IP)(ip)
	}
	return nil
}
