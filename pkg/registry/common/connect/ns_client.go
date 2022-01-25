// Copyright (c) 2021-2022 Cisco and/or its affiliates.
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

package connect

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
)

type connectNSClient struct{}

func (n *connectNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errNoCCProvided
	}
	return registry.NewNetworkServiceRegistryClient(cc).Register(ctx, in, opts...)
}

func (n *connectNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errNoCCProvided
	}
	return registry.NewNetworkServiceRegistryClient(cc).Find(ctx, in, opts...)
}

func (n *connectNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	cc, loaded := clientconn.Load(ctx)
	if !loaded {
		return nil, errNoCCProvided
	}
	return registry.NewNetworkServiceRegistryClient(cc).Unregister(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - returns a new null client that does nothing but call next.NetworkServiceRegistryClient(ctx).
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return new(connectNSClient)
}
