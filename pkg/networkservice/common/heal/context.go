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
	requestHealFuncKey contextKeyType = "requestHealFuncKey"
)

type contextKeyType string

// requestHealFuncType - function used to inform heal server that connection should be healed
//                       - conn              - connection to be healed
//                       - restoreConnection - flag to specify whether what to heal.
//                                             When false, only heals this particular connection.
//                                             When true also tries to heal physical connection to next hop.
type requestHealFuncType func(conn *networkservice.Connection, restoreConnection bool)

func withRequestHealFunc(parent context.Context, fun requestHealFuncType) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, requestHealFuncKey, fun)
}

func requestHealFunc(ctx context.Context) requestHealFuncType {
	if rv, ok := ctx.Value(requestHealFuncKey).(requestHealFuncType); ok {
		return rv
	}
	return nil
}
