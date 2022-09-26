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

// Package updatepath provides a chain element that sets the id of an incoming or outgoing request
package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type updatePathNSClient struct {
	name string
}

// NewNetworkServiceRegistryClient - creates a new updatePath client to update NetworkService path.
func NewNetworkServiceRegistryClient(name string) registry.NetworkServiceRegistryClient {
	return &updatePathNSClient{
		name: name,
	}
}

func (s *updatePathNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	path, index, err := updatePath(in.Path, s.name)
	if err != nil {
		return nil, err
	}

	in.Path = path
	in, err = next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
	path.Index = index

	return in, err
}

func (s *updatePathNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (s *updatePathNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	path, _, err := updatePath(in.Path, s.name)
	if err != nil {
		return nil, err
	}
	in.Path = path

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, in)
}
