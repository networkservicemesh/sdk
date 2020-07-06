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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type nseDoneServer struct{}

func (d *nseDoneServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	markDone(ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (d *nseDoneServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	markDone(server.Context())
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (d *nseDoneServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	markDone(ctx)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

var _ registry.NetworkServiceEndpointRegistryServer = &nseDoneServer{}
