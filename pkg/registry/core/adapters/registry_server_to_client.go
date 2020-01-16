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

// Package adapters provides adapters to translate between registry.{Registry,Discover}{Server,Client}
package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/networkservicemesh/controlplane/api/registry"
)

type registryServerToClient struct {
	server registry.NetworkServiceRegistryServer
}

// NewRegistryServerToClient - returns a new registry.NetworkServiceRegistryServer that is a wrapper around server
func NewRegistryServerToClient(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryClient {
	return &registryServerToClient{server: server}
}

func (r *registryServerToClient) RegisterNSE(ctx context.Context, registration *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	return r.server.RegisterNSE(ctx, registration)
}

func (r *registryServerToClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	return r.server.RemoveNSE(ctx, request)
}
