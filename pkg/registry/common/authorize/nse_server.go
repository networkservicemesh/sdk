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
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/spire"
)

type authorizeNSEServer struct {
	registerPolicies   policiesList
	unregisterPolicies policiesList

	// TODO(nikita): use stringset instead of []string in this map
	spiffeIDNSEsMap *spiffeIDResourcesMap
}

// NewNetworkServiceEndpointRegistryServer - returns a new authorization registry.NetworkServiceEndpointRegistryServer
// Authorize registry server checks spiffeID of NSE.
func NewNetworkServiceEndpointRegistryServer(opts ...Option) registry.NetworkServiceEndpointRegistryServer {
	o := &options{
		registerPolicies:     policiesList{},
		unregisterPolicies:   policiesList{},
		spiffeIDResourcesMap: new(spiffeIDResourcesMap),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSEServer{
		registerPolicies:   o.registerPolicies,
		unregisterPolicies: o.unregisterPolicies,
		spiffeIDNSEsMap:    o.spiffeIDResourcesMap,
	}
}

func (s *authorizeNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	// TODO(nikita): What if we don't have spiffeID ???
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	}

	rawMap := make(map[string][]string)
	s.spiffeIDNSEsMap.Range(func(key spiffeid.ID, value []string) bool {
		rawMap[key.String()] = value
		return true
	})

	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         nse.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.registerPolicies.check(ctx, input); err != nil {
		return nil, err
	}

	nseList, ok := s.spiffeIDNSEsMap.Load(spiffeID)
	if !ok {
		nseList = make([]string, 0)
	}
	nseList = append(nseList, nse.Name)
	s.spiffeIDNSEsMap.Store(spiffeID, nseList)

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *authorizeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	// TODO(nikita): What if we don't have spiffeID ???
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
	}

	rawMap := make(map[string][]string)
	s.spiffeIDNSEsMap.Range(func(key spiffeid.ID, value []string) bool {
		rawMap[key.String()] = value
		return true
	})

	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         nse.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.unregisterPolicies.check(ctx, input); err != nil {
		return nil, err
	}

	// TODO(nikita): What if we are trying to unregister nse that wasn't registered before?
	nseNames, ok := s.spiffeIDNSEsMap.Load(spiffeID)
	if ok {
		for i, nseName := range nseNames {
			if nseName == nse.Name {
				nseNames = append(nseNames[:i], nseNames[i+1])
				break
			}
		}
		if len(nseNames) == 0 {
			s.spiffeIDNSEsMap.Delete(spiffeID)
		} else {
			s.spiffeIDNSEsMap.Store(spiffeID, nseNames)
		}
	}

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
