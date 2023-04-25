// Copyright (c) 2020-2023 Cisco Systems, Inc.
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

// Package trace provides a wrapper for tracing around a dnsutils.Handler
package trace

import (
	"context"

	"github.com/miekg/dns"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
)

type contextKeyType string

const (
	traceInfoKey contextKeyType = "MessageInfo"
	loggedType   string         = "dnsServer"
)

type traceInfo struct {
	RequestMsg  *dns.Msg
	ResponseMsg *dns.Msg
}

func withLog(parent context.Context, operation, methodName, messageID string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	if log.IsTracingEnabled() {
		fields := []*log.Field{log.NewField("type", loggedType), log.NewField("id", messageID)}

		ctx, sLogger, span, sFinish := spanlogger.FromContext(parent, operation, methodName, fields)
		ctx, lLogger, lFinish := logruslogger.FromSpan(ctx, span, operation, fields)
		return withTrace(log.WithLog(ctx, sLogger, lLogger)), func() {
			sFinish()
			lFinish()
		}
	}
	return log.WithLog(parent), func() {}
}

func withTrace(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if _, ok := trace(parent); ok {
		return parent
	}

	return context.WithValue(parent, traceInfoKey, &traceInfo{})
}

func trace(ctx context.Context) (*traceInfo, bool) {
	val, ok := ctx.Value(traceInfoKey).(*traceInfo)
	return val, ok
}
