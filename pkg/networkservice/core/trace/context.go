// Copyright (c) 2020-2022 Cisco Systems, Inc.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package trace provides a wrapper for tracing around a networkservice.NetworkServiceClient
package trace

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
)

type contextKeyType string

const (
	traceInfoKey contextKeyType = "ConnectionInfo"
	loggedType   string         = "networkService"
)

// ConnectionInfo - struct, containing string representations of request and response, used for tracing.
type traceInfo struct {
	// Request is last request of NetworkService{Client, Server}
	Request proto.Message
	// Response is last response of NetworkService{Client, Server}
	Response proto.Message
}

// withLog - provides corresponding logger in context
func withLog(parent context.Context, operation, connectionID string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	// Update outgoing grpc context
	parent = grpcutils.PassTraceToOutgoing(parent)

	if grpcTraceState := grpcutils.TraceFromContext(parent); (grpcTraceState == grpcutils.TraceOn) ||
		(grpcTraceState == grpcutils.TraceUndefined && log.IsTracingEnabled()) {
		ctx, sLogger, span, sFinish := spanlogger.FromContext(parent, operation, map[string]interface{}{"type": loggedType, "id": connectionID})
		ctx, lLogger, lFinish := logruslogger.FromSpan(ctx, span, operation, map[string]interface{}{"type": loggedType, "id": connectionID})
		return withTrace(log.WithLog(ctx, sLogger, lLogger)), func() {
			sFinish()
			lFinish()
		}
	}
	return log.WithLog(parent), func() {}
}

// withConnectionInfo - Provides a traceInfo in context
func withTrace(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if _, ok := trace(parent); ok {
		// We already had connection info
		return parent
	}

	return context.WithValue(parent, traceInfoKey, &traceInfo{})
}

// ConnectionInfo - return traceInfo from context
func trace(ctx context.Context) (*traceInfo, bool) {
	val, ok := ctx.Value(traceInfoKey).(*traceInfo)
	return val, ok
}
