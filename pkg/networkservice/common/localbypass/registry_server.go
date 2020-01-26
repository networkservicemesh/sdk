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

package localbypass

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/peer"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type localRegistry struct {
	server *localBypassServer
}

func newRegistryServer(server *localBypassServer) registry.NetworkServiceRegistryServer {
	return &localRegistry{server: server}
}

func (n *localRegistry) RegisterNSE(ctx context.Context, request *registry.NSERegistration) (*registry.NSERegistration, error) {
	p, ok := peer.FromContext(ctx)
	if ok && p.Addr.Network() == "unix" && n.server != nil {
		u := &url.URL{
			Scheme: "unix",
			Path:   p.Addr.String(),
		}
		n.server.sockets.LoadOrStore(request.GetNetworkServiceEndpoint(), u)
	}
	return next.RegistryServer(ctx).RegisterNSE(ctx, request)
}

func (n *localRegistry) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest) (*empty.Empty, error) {
	if n.server != nil {
		n.server.sockets.Delete(request.GetNetworkServiceEndpointName())
	}
	return next.RegistryServer(ctx).RemoveNSE(ctx, request)
}
