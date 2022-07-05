// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

// Package authorize provides authz checks for incoming or returning connections.
package authorize

import (
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/monitor/next"
)

type authorizeServer struct {
	policies policiesList
}

// NewMonitorConnectionsServer - returns a new authorization networkservicemesh.MonitorConnectionServer
// Authorize server checks left side of Path.
func NewMonitorConnectionsServer(opts ...Option) networkservice.MonitorConnectionServer {
	// todo: add policies
	var s = &authorizeServer{
		policies: []Policy{},
	}
	for _, o := range opts {
		o.apply(&s.policies)
	}
	return s
}

func (a *authorizeServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	// call smth to get ID of the Server/connection
	ctx := srv.Context()
	var leftSide = &networkservice.Path{
		PathSegments: in.GetPathSegments(),
	}
	if _, ok := peer.FromContext(ctx); ok {
		if err := a.policies.check(ctx, leftSide); err != nil {
			return err
		}
	}
	return next.MonitorConnectionServer(ctx).MonitorConnections(in, srv)
}
