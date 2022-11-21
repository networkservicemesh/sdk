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
package authorize

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
)

type authorizeNSEServer struct {
	policies      policiesList
	nsePathIdsMap *ResourcePathIdsMap
}

// NewNetworkServiceEndpointRegistryServer - returns a new authorization registry.NetworkServiceEndpointRegistryServer
// Authorize registry server checks spiffeID of NSE.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	o := &options{
		policies: policiesList{
			opa.WithTokensValidPolicy(),
			opa.WithPrevTokenSignedPolicy(),
			opa.WithTokensExpiredPolicy(),
			opa.WithTokenChainPolicy(),
			opa.WithRegistryClientAllowedPolicy(),
		},
		resourcePathIdsMap: new(ResourcePathIdsMap),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSEServer{
		policies:      o.policies,
		nsePathIdsMap: o.resourcePathIdsMap,
	}
}

func (s *authorizeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if len(s.policies) == 0 {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}
	spiffeID, err := getSpiffeIDFromPath(path)
	if err != nil {
		return nil, err
	}

	index := path.Index
	var leftSide = &grpcmetadata.Path{
		Index:        index,
		PathSegments: path.PathSegments[:index+1],
	}

	rawMap := getRawMap(s.nsePathIdsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       nse.Name,
		ResourcePathIdsMap: rawMap,
		PathSegments:       leftSide.PathSegments,
		Index:              leftSide.Index,
	}

	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	s.nsePathIdsMap.Store(nse.Name, nse.PathIds)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *authorizeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if len(s.policies) == 0 {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	path, err := grpcmetadata.PathFromContext(ctx)
	if err != nil {
		return nil, err
	}
	spiffeID, err := getSpiffeIDFromPath(path)
	if err != nil {
		return nil, err
	}

	index := path.Index
	var leftSide = &grpcmetadata.Path{
		Index:        index,
		PathSegments: path.PathSegments[:index+1],
	}

	rawMap := getRawMap(s.nsePathIdsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       nse.Name,
		ResourcePathIdsMap: rawMap,
		PathSegments:       leftSide.PathSegments,
		Index:              leftSide.Index,
	}

	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	s.nsePathIdsMap.Delete(nse.Name)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
