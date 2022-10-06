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

func (s *updatePathNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if nse.Path == nil {
		nse.Path = &registry.Path{}
	}

	log.FromContext(ctx).Infof("UPDATEPATH [SERVER] INDEX BEFORE REQUEST: %d", nse.Path.Index)

	path, index, err := updatePath(nse.Path, s.name)
	if err != nil {
		return nil, err
	}

	nse.Path = path
	nse, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	nse.Path.Index = index

	log.FromContext(ctx).Infof("UPDATEPATH [SERVER] INDEX AFTER REQUEST: %d", path.Index)

	return nse, err
}

func (s *updatePathNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *updatePathNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	path, _, err := updatePath(nse.Path, s.name)
	if err != nil {
		return nil, err
	}
	nse.Path = path

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
