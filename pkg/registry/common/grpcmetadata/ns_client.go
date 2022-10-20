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

// Package authorize provides authz checks for incoming or returning connections.
package grpcmetadata

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type grpcMetadataNSClient struct {
	nsPathMap *resourcePathMap
}

func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return &grpcMetadataNSClient{
		nsPathMap: new(resourcePathMap),
	}
}

func (c *grpcMetadataNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	path, loaded := c.nsPathMap.Load(ns.Name)
	if !loaded {
		ctxPath, err := PathFromContext(ctx)
		if err != nil {
			return nil, err
		}
		path = ctxPath
	}

	ctx, err := appendToMetadata(ctx, path)
	if err != nil {
		return nil, err
	}

	var header metadata.MD
	opts = append(opts, grpc.Header(&header))
	resp, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	newpath, err := loadFromMetadata(header)
	if err != nil {
		return nil, err
	}

	path.Index = newpath.Index
	path.PathSegments = newpath.PathSegments

	c.nsPathMap.Store(ns.Name, path)

	return resp, nil
}

func (c *grpcMetadataNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *grpcMetadataNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}
