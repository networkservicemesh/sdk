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

package monitor

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

const (
	monitorServerKey contextKeyType = "Server"
)

type contextKeyType string

// WithServer -
//    Wraps 'parent' in a new Context that has the monitor server
func WithServer(parent context.Context, monitor networkservice.MonitorConnection_MonitorConnectionsServer) context.Context {
	if parent == nil {
		parent = context.TODO()
	}
	return context.WithValue(parent, monitorServerKey, monitor)
}

// Server -
//   Returns the Monitor Server
func Server(ctx context.Context) networkservice.MonitorConnection_MonitorConnectionsServer {
	if rv, ok := ctx.Value(monitorServerKey).(networkservice.MonitorConnection_MonitorConnectionsServer); ok {
		return rv
	}
	return nil
}
