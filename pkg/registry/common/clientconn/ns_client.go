// Copyright (c) 2022 Cisco and/or its affiliates.
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

// Package clientconn - chain element for injecting a grpc.ClientConnInterface into the client chain
package clientconn

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type clientConnNSClient struct {
	stringCCMap
}

func (c *clientConnNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx = withClientConnMetadata(ctx, &c.stringCCMap, in.GetName())
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *clientConnNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = withClientConnMetadata(ctx, &c.stringCCMap, in.GetName())
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in)
}

func (c *clientConnNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ctx = withClientConnMetadata(ctx, &c.stringCCMap, uuid.New().String())
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - returns a new null client that does nothing but call next.NetworkServiceRegistryClient(ctx).
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return new(clientConnNSClient)
}
