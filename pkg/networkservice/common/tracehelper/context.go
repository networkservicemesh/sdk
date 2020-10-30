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

package tracehelper

import (
	"context"

	"google.golang.org/protobuf/proto"
)

const (
	connectionInfoKey contextKeyType = "ConnectionInfo"
)

type contextKeyType string

// ConnectionInfo - struct, containing string representations of request and response, used for tracing.
type ConnectionInfo struct {
	// Request is last request of NetworkService{Client, Server}
	Request proto.Message
	// Response is last response of NetworkService{Client, Server}
	Response proto.Message
}

// withConnectionInfo - Provides a ConnectionInfo in context
func withConnectionInfo(parent context.Context, connectionInfo *ConnectionInfo) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	return context.WithValue(parent, connectionInfoKey, connectionInfo)
}

// FromContext - return ConnectionInfo from context
func FromContext(ctx context.Context) (*ConnectionInfo, bool) {
	val, ok := ctx.Value(connectionInfoKey).(*ConnectionInfo)
	return val, ok
}
