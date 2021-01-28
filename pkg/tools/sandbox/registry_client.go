// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package sandbox

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice/payload"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/interpose"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

type registryClient struct {
	nsClient  registry.NetworkServiceRegistryClient
	nseClient registry.NetworkServiceEndpointRegistryClient
}

// NewRegistryClient returns NSE registry client to register NSE to registry
func NewRegistryClient(ctx context.Context, cc grpc.ClientConnInterface, isForwarder bool) registry.NetworkServiceEndpointRegistryClient {
	var interposeClient registry.NetworkServiceEndpointRegistryClient
	if isForwarder {
		interposeClient = interpose.NewNetworkServiceEndpointRegistryClient()
	} else {
		interposeClient = null.NewNetworkServiceEndpointRegistryClient()
	}
	return &registryClient{
		nsClient: registry.NewNetworkServiceRegistryClient(cc),
		nseClient: chain.NewNetworkServiceEndpointRegistryClient(
			refresh.NewNetworkServiceEndpointRegistryClient(
				refresh.WithChainContext(ctx)),
			interposeClient,
			registry.NewNetworkServiceEndpointRegistryClient(cc),
		),
	}
}

func (c *registryClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (_ *registry.NetworkServiceEndpoint, err error) {
	if nse, err = c.nseClient.Register(ctx, nse, opts...); err != nil {
		return nil, err
	}
	for _, name := range nse.NetworkServiceNames {
		if _, err = c.nsClient.Register(ctx, &registry.NetworkService{
			Name:    name,
			Payload: payload.IP,
		}, opts...); err != nil {
			return nil, err
		}
	}
	return nse, err
}

func (c *registryClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return c.nseClient.Find(ctx, query, opts...)
}

func (c *registryClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (_ *empty.Empty, err error) {
	if _, unregisterErr := c.nseClient.Unregister(ctx, nse, opts...); unregisterErr != nil {
		err = errors.Wrapf(unregisterErr, "%v\n", err)
	}
	for _, name := range nse.NetworkServiceNames {
		if _, unregisterErr := c.nsClient.Unregister(ctx, &registry.NetworkService{
			Name:    name,
			Payload: payload.IP,
		}, opts...); unregisterErr != nil {
			err = errors.Wrapf(unregisterErr, "%v\n", err)
		}
	}
	return new(empty.Empty), err
}
