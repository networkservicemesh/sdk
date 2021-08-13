// Copyright (c) 2020 Cisco Systems, Inc.
//
// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
)

type nextServer struct {
	nextParent networkservice.NetworkServiceServer
	servers    []networkservice.NetworkServiceServer
	index      int
}

// ServerWrapper - A function that wraps a networkservice.NetworkServiceServer
type ServerWrapper func(networkservice.NetworkServiceServer) networkservice.NetworkServiceServer

// ServerChainer - A function that chains a list of networkservice.NetworkServiceServers together
type ServerChainer func(...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer

// NewWrappedNetworkServiceServer - chains together the servers provides with the wrapper wrapped around each one in turn.
func NewWrappedNetworkServiceServer(wrapper ServerWrapper, servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	rv := &nextServer{servers: make([]networkservice.NetworkServiceServer, 0, len(servers))}
	for _, s := range servers {
		rv.servers = append(rv.servers, wrapper(s))
	}
	return rv
}

// NewNetworkServiceServer - chains together servers while providing them with the correct next.Server(ctx) to call to
// invoke the next element in the chain.
func NewNetworkServiceServer(servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return NewWrappedNetworkServiceServer(func(server networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
		return server
	}, servers...)
}

func (n *nextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	server, ctx := n.getServerAndContext(ctx)
	return server.Request(ctx, request)
}

func (n *nextServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	server, ctx := n.getServerAndContext(ctx)
	return server.Close(ctx, conn)
}

func (n *nextServer) getServerAndContext(ctx context.Context) (networkservice.NetworkServiceServer, context.Context) {
	nextParent := n.nextParent
	if n.index == 0 {
		nextParent = Server(ctx)
		if len(n.servers) == 0 {
			return nextParent, ctx
		}
	}
	if n.index+1 < len(n.servers) {
		return n.servers[n.index], withNextServer(ctx, &nextServer{nextParent: nextParent, servers: n.servers, index: n.index + 1})
	}
	return n.servers[n.index], withNextServer(ctx, nextParent)
}
