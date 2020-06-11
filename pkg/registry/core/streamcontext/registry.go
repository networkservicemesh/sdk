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

package streamcontext

import (
	"context"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

type networkServiceRegistryFindClient struct {
	registry.NetworkServiceRegistry_FindClient
	ctx context.Context
}

func (s *networkServiceRegistryFindClient) Context() context.Context {
	return s.ctx
}

func NetworkServiceRegistryFindClient(ctx context.Context, client registry.NetworkServiceRegistry_FindClient) registry.NetworkServiceRegistry_FindClient {
	return &networkServiceRegistryFindClient{
		ctx:                               extend.WithValuesFromContext(ctx, client.Context()),
		NetworkServiceRegistry_FindClient: client,
	}
}

type networkServiceRegistryFindServer struct {
	registry.NetworkServiceRegistry_FindServer
	ctx context.Context
}

func (s *networkServiceRegistryFindServer) Context() context.Context {
	return s.ctx
}

func NetworkServiceRegistryFindServer(ctx context.Context, server registry.NetworkServiceRegistry_FindServer) registry.NetworkServiceRegistry_FindServer {
	return &networkServiceRegistryFindServer{
		ctx:                               extend.WithValuesFromContext(ctx, server.Context()),
		NetworkServiceRegistry_FindServer: server,
	}
}
