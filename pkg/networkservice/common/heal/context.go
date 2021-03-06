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

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

const (
	requestHealConnectionFuncKey    contextKeyType = "requestHealConnectionFuncKey"
	requestRestoreConnectionFuncKey contextKeyType = "requestRestoreConnectionFuncKey"
)

type contextKeyType string

type requestHealFuncType func(conn *networkservice.Connection)

func withRequestHealConnectionFunc(parent context.Context, fun requestHealFuncType) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestHealConnectionFuncKey, fun)
}

func requestHealConnectionFunc(ctx context.Context) requestHealFuncType {
	if rv, ok := ctx.Value(requestHealConnectionFuncKey).(requestHealFuncType); ok {
		return rv
	}
	return nil
}

func withRequestRestoreConnectionFunc(parent context.Context, fun requestHealFuncType) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestRestoreConnectionFuncKey, fun)
}

func requestRestoreConnectionFunc(ctx context.Context) requestHealFuncType {
	if rv, ok := ctx.Value(requestRestoreConnectionFuncKey).(requestHealFuncType); ok {
		return rv
	}
	return nil
}
