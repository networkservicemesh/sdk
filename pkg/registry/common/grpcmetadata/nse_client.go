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

package grpcmetadata

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type grpcMetadataNSEClient struct{}

// NewNetworkServiceEndpointRegistryClient - returns grpcmetadata NSE client that sends metadata to server and receives it back.
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return &grpcMetadataNSEClient{}
}

func (c *grpcMetadataNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	path := PathFromContext(ctx)

	ctx, err := appendToMetadata(ctx, path)
	if err != nil {
		return nil, err
	}

	var header metadata.MD
	opts = append(opts, grpc.Header(&header))
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}

	newpath, err := fromMD(header)

	if err == nil {
		path.Index = newpath.Index
		path.PathSegments = newpath.PathSegments
	}

	return resp, nil
}

func (c *grpcMetadataNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *grpcMetadataNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	path := PathFromContext(ctx)

	ctx, err := appendToMetadata(ctx, path)
	if err != nil {
		return nil, err
	}

	var header metadata.MD
	opts = append(opts, grpc.Header(&header))

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}

	newpath, err := fromMD(header)
	if err == nil {
		path.Index = newpath.Index
		path.PathSegments = newpath.PathSegments
	}
	return resp, nil
}
