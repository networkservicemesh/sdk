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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type grpcMetadataNSEServer struct{}

// NewNetworkServiceEndpointRegistryServer - returns grpcmetadata NSE server that receives metadata from client and sends it back.
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &grpcMetadataNSEServer{}
}

func (s *grpcMetadataNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	path, err := fromContext(ctx)
	if err != nil {
		log.FromContext(ctx).Warnf("Register: failed to load grpc metadata from context: %v", err.Error())
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	ctx = PathWithContext(ctx, path)

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	err = sendPath(ctx, path)

	return resp, err
}

func (s *grpcMetadataNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *grpcMetadataNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	path, err := fromContext(ctx)
	if err != nil {
		log.FromContext(ctx).Warnf("Unegister: failed to load grpc metadata from context: %v", err.Error())
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	ctx = PathWithContext(ctx, path)

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	if err != nil {
		return nil, err
	}

	err = sendPath(ctx, path)

	return resp, err
}
