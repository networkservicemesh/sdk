// Copyright (c) 2020-2024 Doc.ai and/or its affiliates.
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

// Package interdomainbypass provides a chain element that manages registered NSEs
package interdomainbypass

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type interdomainBypassFindServer struct {
	value string
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *interdomainBypassFindServer) Send(resp *registry.NetworkServiceEndpointResponse) error {
	resp.GetNetworkServiceEndpoint().Url = s.value
	return s.NetworkServiceEndpointRegistry_FindServer.Send(resp)
}

type interdomainBypassServer struct {
	nseToNsmgrMap genericsync.Map[string, string]
}

func (n *interdomainBypassServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	n.nseToNsmgrMap.Store(service.GetName(), service.GetUrl())
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (n *interdomainBypassServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	if v, ok := n.nseToNsmgrMap.Load(query.GetNetworkServiceEndpoint().GetName()); ok {
		server = &interdomainBypassFindServer{
			NetworkServiceEndpointRegistry_FindServer: server,
			value: v,
		}
	}

	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (n *interdomainBypassServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	n.nseToNsmgrMap.Delete(service.GetName())

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(interdomainBypassServer)
}
