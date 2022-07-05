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

package next

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type nextMonitorConnectionServer struct {
	servers    []networkservice.MonitorConnectionServer
	index      int
	nextParent networkservice.MonitorConnectionServer
}

// MonitorConnectionsServerWrapper - a function that wraps around a networkservice.MonitorConnectionServer
type MonitorConnectionsServerWrapper func(networkservice.MonitorConnectionServer) networkservice.MonitorConnectionServer

// MonitorConnectionsServerChainer - a function that chains together a list of networkservice.MonitorConnectionServers
type MonitorConnectionsServerChainer func(...networkservice.MonitorConnectionServer) networkservice.MonitorConnectionServer

func (n *nextMonitorConnectionServer) MonitorConnections(in *networkservice.MonitorScopeSelector, srv networkservice.MonitorConnection_MonitorConnectionsServer) error {
	server, _ := n.getServerAndContext(srv.Context())
	return server.MonitorConnections(in, srv)
}

func (n *nextMonitorConnectionServer) getServerAndContext(ctx context.Context) (networkservice.MonitorConnectionServer, context.Context) {
	nextParent := n.nextParent
	if n.index == 0 {
		nextParent = MonitorConnectionServer(ctx)
		if len(n.servers) == 0 {
			return nextParent, ctx
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index], withNextMonitorConnectionServer(ctx, &nextMonitorConnectionServer{nextParent: nextParent, servers: n.servers, index: n.index + 1})
	}
	return n.servers[n.index], withNextMonitorConnectionServer(ctx, nextParent)
}
