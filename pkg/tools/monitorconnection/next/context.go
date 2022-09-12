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

// Package next provides a mechanism for chained networkservice.MonitorConnection{Server,Client}s to call
// the next element in the chain.
package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type contextKeyType string

const (
	nextMonitorConnectionServerKey contextKeyType = "NextMonitorConnectionServer"
)

// withNextMonitorConnectionServer -
//
//	Wraps 'parent' in a new Context that has the Server networkservice.MonitorConnectionServer to be called in the chain
func withNextMonitorConnectionServer(parent context.Context, next networkservice.MonitorConnectionServer) context.Context {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithValue(parent, nextMonitorConnectionServerKey, next)
}

// MonitorConnectionServer -
//
//	Returns the networkservice.MonitorConnectionServer to be called in the chain from the context.Context
func MonitorConnectionServer(ctx context.Context) networkservice.MonitorConnectionServer {
	rv, ok := ctx.Value(nextMonitorConnectionServerKey).(networkservice.MonitorConnectionServer)
	if ok && rv != nil {
		return rv
	}
	return &tailMonitorConnectionsServer{}
}
