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

// Package streamcontext provides API to extend context for find client/server
package streamcontext

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

type networkServiceEndpointRegistryFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
	ctx context.Context
}

func (s *networkServiceEndpointRegistryFindClient) Context() context.Context {
	return s.ctx
}

// NetworkServiceEndpointRegistryFindClient extends context for passed NetworkServiceEndpointRegistry_FindClient
func NetworkServiceEndpointRegistryFindClient(ctx context.Context, client registry.NetworkServiceEndpointRegistry_FindClient) registry.NetworkServiceEndpointRegistry_FindClient {
	if client != nil {
		ctx = extend.WithValuesFromContext(client.Context(), ctx)
	}
	return &networkServiceEndpointRegistryFindClient{
		ctx: ctx,
		NetworkServiceEndpointRegistry_FindClient: client,
	}
}

type networkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	ctx context.Context
}

func (s *networkServiceEndpointRegistryFindServer) Context() context.Context {
	return s.ctx
}

// NetworkServiceEndpointRegistryFindServer extends context for passed NetworkServiceEndpointRegistry_FindServer
func NetworkServiceEndpointRegistryFindServer(ctx context.Context, server registry.NetworkServiceEndpointRegistry_FindServer) registry.NetworkServiceEndpointRegistry_FindServer {
	if server != nil {
		ctx = extend.WithValuesFromContext(server.Context(), ctx)
	}
	return &networkServiceEndpointRegistryFindServer{
		ctx: ctx,
		NetworkServiceEndpointRegistry_FindServer: server,
	}
}
