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

	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type nextMonitorConnectionClient struct {
	clients    []networkservice.MonitorConnectionClient
	index      int
	nextParent networkservice.MonitorConnectionClient
}

// MonitorConnectionClientWrapper - a function that wraps around a networkservice.MonitorConnectionClient
type MonitorConnectionClientWrapper func(networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient

// MonitorConnectionClientChainer - a function that chains together a list of networkservice.MonitorConnectionClient
type MonitorConnectionClientChainer func(...networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient

// NewWrappedMonitorConnectionClient chains together clients with wrapper wrapped around each one
func NewWrappedMonitorConnectionClient(wrapper MonitorConnectionClientWrapper, clients ...networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient {
	rv := &nextMonitorConnectionClient{clients: make([]networkservice.MonitorConnectionClient, 0, len(clients))}
	for _, c := range clients {
		rv.clients = append(rv.clients, wrapper(c))
	}
	return rv
}

// NewMonitorConnectionClient - chains together clients into a single networkservice.MonitorConnectionClient
func NewMonitorConnectionClient(clients ...networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient {
	return NewWrappedMonitorConnectionClient(func(client networkservice.MonitorConnectionClient) networkservice.MonitorConnectionClient {
		return client
	}, clients...)
}

func (n *nextMonitorConnectionClient) MonitorConnections(ctx context.Context, in *networkservice.MonitorScopeSelector, opts ...grpc.CallOption) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	client, ctx := n.getClientAndContext(ctx)
	return client.MonitorConnections(ctx, in, opts...)
}

func (n *nextMonitorConnectionClient) getClientAndContext(ctx context.Context) (networkservice.MonitorConnectionClient, context.Context) {
	nextParent := n.nextParent
	if n.index == 0 {
		nextParent = MonitorConnectionClient(ctx)
		if len(n.clients) == 0 {
			return nextParent, ctx
		}
	}
	if n.index+1 < len(n.clients) {
		return n.clients[n.index], withNextMonitorConnectionClient(ctx, &nextMonitorConnectionClient{nextParent: nextParent, clients: n.clients, index: n.index + 1})
	}
	return n.clients[n.index], withNextMonitorConnectionClient(ctx, nextParent)
}
