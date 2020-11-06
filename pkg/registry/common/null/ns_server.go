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

package null

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type nullNSServer struct{}

func (n *nullNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (n *nullNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (n *nullNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceRegistryServer - returns a new null server that does nothing but call next.NetworkServiceRegistryServer(ctx).
func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return new(nullNSServer)
}
