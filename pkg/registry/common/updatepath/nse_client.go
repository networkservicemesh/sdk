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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type updatePathNSEClient struct {
	name string
}

// NewNetworkServiceEndpointRegistryClient - creates a new updatePath client to update NetworkServiceEndoint path.
func NewNetworkServiceEndpointRegistryClient(name string) registry.NetworkServiceEndpointRegistryClient {
	return &updatePathNSEClient{
		name: name,
	}
}

func (s *updatePathNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	log.FromContext(ctx).Infof("updatepath opts: %v", opts)

	if nse.Path == nil {
		nse.Path = &registry.Path{}
	}

	log.FromContext(ctx).Infof("UPDATEPATH [CLIENT] INDEX BEFORE REQUEST: %d", nse.Path.Index)

	path, index, err := updatePath(nse.Path, s.name)
	if err != nil {
		return nil, err
	}

	nse.Path = path
	nse, err = next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	nse.Path.Index = index

	log.FromContext(ctx).Infof("UPDATEPATH [CLIENT] INDEX AFTER REQUEST: %d", path.Index)

	return nse, err
}

func (s *updatePathNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *updatePathNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	path, _, err := updatePath(nse.Path, s.name)
	if err != nil {
		return nil, err
	}
	nse.Path = path

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
