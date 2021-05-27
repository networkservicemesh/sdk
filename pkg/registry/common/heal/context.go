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

package heal

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

const (
	requestNSERestoreKey     contextKeyType = "requestNSERestore"
	requestNSERestoreFindKey contextKeyType = "requestNSERestoreFind"
	requestNSRestoreKey      contextKeyType = "requestNSRestore"
	requestNSRestoreFindKey  contextKeyType = "requestNSRestoreFind"
)

type contextKeyType string

type requestNSERestoreFunc func(nse *registry.NetworkServiceEndpoint)
type requestNSERestoreFindFunc func()

func withRequestNSERestore(parent context.Context, f requestNSERestoreFunc) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestNSERestoreKey, f)
}

func requestNSERestore(ctx context.Context) requestNSERestoreFunc {
	if rv, ok := ctx.Value(requestNSERestoreKey).(requestNSERestoreFunc); ok {
		return rv
	}
	return func(nse *registry.NetworkServiceEndpoint) {}
}

func withRequestNSERestoreFind(parent context.Context, f requestNSERestoreFindFunc) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestNSERestoreFindKey, f)
}

func requestNSERestoreFind(ctx context.Context) requestNSERestoreFindFunc {
	if rv, ok := ctx.Value(requestNSERestoreFindKey).(requestNSERestoreFindFunc); ok {
		return rv
	}
	return func() {}
}

type requestNSRestoreFunc func(ns *registry.NetworkService)

func withRequestNSRestore(parent context.Context, f requestNSRestoreFunc) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestNSRestoreKey, f)
}

func requestNSRestore(ctx context.Context) requestNSRestoreFunc {
	if rv, ok := ctx.Value(requestNSRestoreKey).(requestNSRestoreFunc); ok {
		return rv
	}
	return func(ns *registry.NetworkService) {}
}
