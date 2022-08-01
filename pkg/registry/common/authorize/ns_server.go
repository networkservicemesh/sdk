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

type authorizeNSServer struct {
	registerPolicies   policiesList
	unregisterPolicies policiesList

	// TODO(nikita): use stringset instead of []string in this map
	spiffeIDNSsMap *spiffeIDResourcesMap
}

// NewNetworkServiceRegistryServer - returns a new authorization registry.NetworkServiceRegistryServer
// Authorize registry server checks spiffeID of NS.
func NewNetworkServiceRegistryServer(opts ...Option) registry.NetworkServiceRegistryServer {
	o := &options{
		registerPolicies:     policiesList{},
		unregisterPolicies:   policiesList{},
		spiffeIDResourcesMap: new(spiffeIDResourcesMap),
	}

	for _, opt := range opts {
		opt(o)
	}

	return &authorizeNSServer{
		registerPolicies:   o.registerPolicies,
		unregisterPolicies: o.unregisterPolicies,
		spiffeIDNSsMap:     o.spiffeIDResourcesMap,
	}
}

func (s *authorizeNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	// TODO(nikita): What if we don't have spiffeID ???
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil {
		return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	}

	rawMap := make(map[string][]string)
	s.spiffeIDNSsMap.Range(func(key spiffeid.ID, value []string) bool {
		rawMap[key.String()] = value
		return true
	})

	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         ns.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.registerPolicies.check(ctx, input); err != nil {
		return nil, err
	}

	nsNames, ok := s.spiffeIDNSsMap.Load(spiffeID)
	if !ok {
		nsNames = make([]string, 0)
	}
	nsNames = append(nsNames, ns.Name)
	s.spiffeIDNSsMap.Store(spiffeID, nsNames)

	return next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
}

func (s *authorizeNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *authorizeNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	// TODO(nikita): What if we don't have spiffeID ???
	spiffeID, err := spire.SpiffeIDFromContext(ctx)
	if err != nil {
		return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
	}

	rawMap := make(map[string][]string)
	s.spiffeIDNSsMap.Range(func(key spiffeid.ID, value []string) bool {
		rawMap[key.String()] = value
		return true
	})

	input := RegistryOpaInput{
		SpiffeID:             spiffeID.String(),
		ResourceName:         ns.Name,
		SpiffeIDResourcesMap: rawMap,
	}
	if err := s.unregisterPolicies.check(ctx, input); err != nil {
		return nil, err
	}

	// TODO(nikita): What if we are trying to unregister nse that wasn't registered before?
	nsNames, ok := s.spiffeIDNSsMap.Load(spiffeID)
	if ok {
		for i, nsName := range nsNames {
			if nsName == ns.Name {
				nsNames = append(nsNames[:i], nsNames[i+1])
				break
			}
		}
	}
	s.spiffeIDNSsMap.Store(spiffeID, nsNames)

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
