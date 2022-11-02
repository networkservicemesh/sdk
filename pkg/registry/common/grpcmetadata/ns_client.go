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

// Package grpcmetadata provides chain elements that transfer grpc metadata between server and client.
package grpcmetadata

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type grpcMetadataNSClient struct {
}

// NewNetworkServiceRegistryClient - returns grpcmetadata NS client that sends metadata to server and receives it back
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return &grpcMetadataNSClient{}
}

func printPath(ctx context.Context, path *registry.Path) {
	logger := log.FromContext(ctx)

	for i, s := range path.PathSegments {
		logger.Infof("Segment: %d, Expires: %v, Value: %v", i, s.Expires, s)
	}
}

func (c *grpcMetadataNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	path, err := PathFromContext(ctx)
	if err != nil {
		return nil, err
	}
	log.FromContext(ctx).Infof("GRPCMETADATA CLIENT MAP")
	printPath(ctx, path)
	ctx, err = appendToMetadata(ctx, path)
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

	// c.nsPathMap.Store(ns.Name, path)

	return resp, nil
}

func (c *grpcMetadataNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *grpcMetadataNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	path, err := PathFromContext(ctx)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Infof("GRPCMETADATA CLIENT MAP")
	log.FromContext(ctx).Infof("INDEX: %v", path.Index)
	printPath(ctx, path)
	ctx, err = appendToMetadata(ctx, path)
	if err != nil {
		return nil, err
	}

	resp, err := next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
