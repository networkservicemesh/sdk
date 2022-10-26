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

package updatepath

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
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

func (s *updatePathNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		path = &registry.Path{}
		ctx = grpcmetadata.PathWithContext(ctx, path)
	}

	path, index, err := updatePath(path, s.name)
	if err != nil {
		return nil, err
	}

	ctx = grpcmetadata.PathWithContext(ctx, path)

	ns, err = next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}
	path.Index = index

	return ns, err
}

func (s *updatePathNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *updatePathNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}

	path, index, err := updatePath(path, s.name)
	if err != nil {
		return nil, err
	}

	resp, err := next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	path.Index = index
	return resp, err
}
