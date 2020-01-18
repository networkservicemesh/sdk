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

// Package localbypass provides a NetworkServiceServer chain element that tracks local Endpoints and substitutes
// their unix file socket as the clienturl.ClientURL(ctx) used to connect to them.
package localbypass

import (
	"context"
	"net/url"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/connection"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/networkservice"
	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type localBypassServer struct {
	// Map of names -> *url.URLs for local bypass to file sockets
	sockets sync.Map
}

// NewServer - creates a NetworkServiceServer that tracks locally registered Endpoints substitutes their
//             unix file socket as the clienturl.ClientURL(ctx) used to connect to them.
//             - server - *registry.NetworkServiceRegistryServer.  Since registry.NetworkServiceRegistryServer is an interface
//                        (and thus a pointer) *registry.NetworkServiceRegistryServer is a double pointer.  Meaning it
//                        points to a place that points to a place that implements registry.NetworkServiceRegistryServer
//                        This is done so that we can return a registry.NetworkServiceRegistryServer chain element
//                        while maintaining the NewServer pattern for use like anything else in a chain.
//                        The value in *server must be included in the registry.NetworkServiceRegistryServer listening
//                        so it can capture the registrations.
func NewServer(server *registry.NetworkServiceRegistryServer) networkservice.NetworkServiceServer {
	rv := &localBypassServer{}
	*server = newRegistryServer(rv)
	return rv
}

func (l *localBypassServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*connection.Connection, error) {
	if v, ok := l.sockets.Load(request.GetConnection().GetNetworkServiceEndpointName()); ok && v != nil {
		if u, ok := v.(*url.URL); ok {
			ctx = clienturl.WithClientURL(ctx, u)
		}
	}
	return next.Server(ctx).Request(ctx, request)
}

func (l *localBypassServer) Close(ctx context.Context, conn *connection.Connection) (*empty.Empty, error) {
	if v, ok := l.sockets.Load(conn.GetNetworkServiceEndpointName()); ok && v != nil {
		if u, ok := v.(*url.URL); ok {
			ctx = clienturl.WithClientURL(ctx, u)
		}
	}
	return next.Server(ctx).Close(ctx, conn)
}
