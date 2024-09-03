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

const pathKey = "path"

type grpcMetadataNSServer struct{}

// NewNetworkServiceRegistryServer - returns grpcmetadata NS server that receives metadata from client and sends it back.
func NewNetworkServiceRegistryServer() registry.NetworkServiceRegistryServer {
	return &grpcMetadataNSServer{}
}

func (s *grpcMetadataNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	path, err := fromContext(ctx)
	if err != nil {
		log.FromContext(ctx).Warnf("Register: failed to load grpc metadata from context: %v", err.Error())
		return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	}

	ctx = PathWithContext(ctx, path)

	resp, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}

	err = sendPath(ctx, path)

	return resp, err
}

func (s *grpcMetadataNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *grpcMetadataNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	path, err := fromContext(ctx)
	if err != nil {
		log.FromContext(ctx).Warnf("Unregister: failed to load grpc metadata from context: %v", err.Error())
		return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	}

	ctx = PathWithContext(ctx, path)

	resp, err := next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	if err != nil {
		return nil, err
	}

	err = sendPath(ctx, path)

	return resp, err
}
