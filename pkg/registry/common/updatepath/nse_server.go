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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type updatePathNSEServer struct {
	name string
}

// NewNetworkServiceEndpointRegistryServer - creates a new updatePath server to update NetworkServiceEndpoint path.
func NewNetworkServiceEndpointRegistryServer(name string) registry.NetworkServiceEndpointRegistryServer {
	return &updatePathNSEServer{
		name: name,
	}
}

func printPath(ctx context.Context, path *registry.Path) {
	logger := log.FromContext(ctx)

	for i, s := range path.PathSegments {
		logger.Infof("Segment: %d, Expires: %v, Value: %v", i, s.Expires, s)
	}
}

func (s *updatePathNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Info("UPDATE PATH SERVER PATH BEFORE UPDATE")
	log.FromContext(ctx).Infof("INDEX: %v", path.Index)
	printPath(ctx, path)
	path, index, err := updatePath(path, s.name)
	if err != nil {
		return nil, err
	}

	log.FromContext(ctx).Info("UPDATE PATH SERVER PATH AFTER UPDATE")
	log.FromContext(ctx).Infof("INDEX: %v", path.Index)
	printPath(ctx, path)

	nse, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	path.Index = index
	return nse, nil
}

func (s *updatePathNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *updatePathNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}

	path, index, err := updatePath(path, s.name)
	if err != nil {
		return nil, err
	}

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	path.Index = index
	return resp, err
}
