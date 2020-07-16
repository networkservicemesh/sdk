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

package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type contextNSServer struct{}

func (c *contextNSServer) Register(ctx context.Context, in *registry.NetworkService) (*registry.NetworkService, error) {
	captureContext(ctx)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, in)
}

func (c *contextNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	captureContext(server.Context())
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (c *contextNSServer) Unregister(ctx context.Context, in *registry.NetworkService) (*empty.Empty, error) {
	captureContext(ctx)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, in)
}

var _ registry.NetworkServiceRegistryServer = &contextNSServer{}
