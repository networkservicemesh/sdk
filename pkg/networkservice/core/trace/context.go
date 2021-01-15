// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package trace provides a wrapper for tracing around a networkservice.NetworkServiceClient
package trace

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"
	"github.com/networkservicemesh/sdk/pkg/tools/logger/tracelogger"

	"google.golang.org/protobuf/proto"
)

type contextKeyType string

const (
	traceInfoKey contextKeyType = "ConnectionInfo"
)

// ConnectionInfo - struct, containing string representations of request and response, used for tracing.
type traceInfo struct {
	// Request is last request of NetworkService{Client, Server}
	Request proto.Message
	// Response is last response of NetworkService{Client, Server}
	Response proto.Message
}

// withLog - provides corresponding logger in context
func withLog(parent context.Context, operation string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	if logger.IsTracingEnabled() {
		return tracelogger.WithLog(withTrace(parent), operation)
	}
	return logger.WithLog(parent), func() {}
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
