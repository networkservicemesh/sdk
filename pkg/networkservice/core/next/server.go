// Copyright (c) 2020 Cisco Systems, Inc.
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

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
)

type nextServer struct {
	servers []networkservice.NetworkServiceServer
	index   int
}

// ServerWrapper - A function that wraps a networkservice.NetworkServiceServer
type ServerWrapper func(networkservice.NetworkServiceServer) networkservice.NetworkServiceServer

// ServerChainer - A function that chains a list of networkservice.NetworkServiceServers together
type ServerChainer func(...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer

// NewWrappedNetworkServiceServer - chains together the servers provides with the wrapper wrapped around each one in turn.
func NewWrappedNetworkServiceServer(wrapper ServerWrapper, servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	rv := &nextServer{
		servers: servers,
	}
	for i := range rv.servers {
		rv.servers[i] = wrapper(rv.servers[i])
	}
	return rv
}

// NewNetworkServiceServer - chains together servers while providing them with the correct next.Server(ctx) to call to
// invoke the next element in the chain.
func NewNetworkServiceServer(servers ...networkservice.NetworkServiceServer) networkservice.NetworkServiceServer {
	return NewWrappedNetworkServiceServer(nil, servers...)
}

func (n *nextServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Request(withNextServer(ctx, &nextServer{servers: n.servers, index: n.index + 1}), request)
	}
	return n.servers[n.index].Request(withNextServer(ctx, newTailServer()), request)
}

func (n *nextServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if n.index+1 < len(n.servers) {
		return n.servers[n.index].Close(withNextServer(ctx, &nextServer{servers: n.servers, index: n.index + 1}), conn)
	}
	return n.servers[n.index].Close(withNextServer(ctx, newTailServer()), conn)
}
