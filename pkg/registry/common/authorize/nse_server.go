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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/opa"
	"github.com/networkservicemesh/sdk/pkg/tools/spire"
	"github.com/networkservicemesh/sdk/pkg/tools/stringset"
)

type authorizeNSEServer struct {
	policies        policiesList
	spiffeIDNSEsMap *spiffeIDResourcesMap
}

// NewNetworkServiceEndpointRegistryServer - returns a new authorization registry.NetworkServiceEndpointRegistryServer
// Authorize registry server checks spiffeID of NSE.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	o := &options{
		policies:             policiesList{opa.WithRegistryClientAllowedPolicy()},
		spiffeIDResourcesMap: new(spiffeIDResourcesMap),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSEServer{
		policies:        o.policies,
		spiffeIDNSEsMap: o.spiffeIDResourcesMap,
	}
}

func (s *authorizeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil && len(s.policies) == 0 {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	rawMap := getRawMap(s.spiffeIDNSEsMap)
	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         nse.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	nseNames, ok := s.spiffeIDNSEsMap.Load(spiffeID)
	if !ok {
		nseNames = new(stringset.StringSet)
	}
	nseNames.LoadOrStore(nse.Name, struct{}{})
	s.spiffeIDNSEsMap.Store(spiffeID, nseNames)

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *authorizeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil && len(s.policies) == 0 {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	rawMap := getRawMap(s.spiffeIDNSEsMap)
	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         nse.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.policies.check(ctx, input); err != nil {
		return nil, err
	}

	nseNames, ok := s.spiffeIDNSEsMap.Load(spiffeID)
	if ok {
		nseNames.Delete(nse.Name)
		namesEmpty := true
		nseNames.Range(func(key string, value struct{}) bool {
			namesEmpty = false
			return true
		})

		if namesEmpty {
			s.spiffeIDNSEsMap.Delete(spiffeID)
		} else {
			s.spiffeIDNSEsMap.Store(spiffeID, nseNames)
		}
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
