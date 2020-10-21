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
	"google.golang.org/grpc"
)

type callNextNSServer struct {
	server registry.NetworkServiceRegistryServer
}

func (c *callNextNSServer) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return c.server.Register(ctx, in)
}

func (c *callNextNSServer) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return nsFindServerToClient(ctx, c.server, in)
}

func (c *callNextNSServer) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return c.server.Unregister(ctx, in)
}

var _ registry.NetworkServiceRegistryClient = &callNextNSServer{}
