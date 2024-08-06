// Copyright (c) 2022-2024 Cisco and/or its affiliates.
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

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type authorizeNSServer struct {
	policies     policiesList
	nsPathIDsMap *genericsync.Map[string, []string]
}

// NewNetworkServiceRegistryServer - returns a new authorization registry.NetworkServiceRegistryServer
// Authorize registry server checks spiffeID of NS.
func NewNetworkServiceRegistryServer(opts ...Option) registry.NetworkServiceRegistryServer {
	o := &options{
		resourcePathIDsMap: new(genericsync.Map[string, []string]),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSServer{
		policies:     o.policies,
		nsPathIDsMap: o.resourcePathIDsMap,
	}
}

func (s *authorizeNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	if len(s.policies) == 0 {
		return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	}

	path := grpcmetadata.PathFromContext(ctx)
	spiffeID := getSpiffeIDFromPath(ctx, path)
	leftSide := getLeftSideOfPath(path)

	rawMap := getRawMap(s.nsPathIDsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       ns.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       leftSide.PathSegments,
		Index:              leftSide.Index,
	}
	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	ns, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}
	s.nsPathIDsMap.Store(ns.Name, ns.PathIds)
	return ns, nil
}

func (s *authorizeNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	if len(s.policies) == 0 {
		return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	}

	path := grpcmetadata.PathFromContext(ctx)
	spiffeID := getSpiffeIDFromPath(ctx, path)
	leftSide := getLeftSideOfPath(path)

	rawMap := getRawMap(s.nsPathIDsMap)
	input := RegistryOpaInput{
		ResourceID:         spiffeID.String(),
		ResourceName:       ns.Name,
		ResourcePathIDsMap: rawMap,
		PathSegments:       leftSide.PathSegments,
		Index:              leftSide.Index,
	}
	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	s.nsPathIDsMap.Delete(ns.Name)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
