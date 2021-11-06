// Copyright (c) 2021 Cisco and/or its affiliates.
//
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

// Package setnetworkservicenames provides registry.NetworkServiceEndpointRegistryClient that simply sets NetworkServiceNames for each incoming registration/find/unregistration.
package setnetworkservicenames

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setNetworkServiceNamesNSEClient struct {
	networkServices []string
}

func (n *setNetworkServiceNamesNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	in.NetworkServiceNames = n.networkServices
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (n *setNetworkServiceNamesNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	in.GetNetworkServiceEndpoint().NetworkServiceNames = n.networkServices
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (n *setNetworkServiceNamesNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	in.NetworkServiceNames = n.networkServices
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient - returns registry.NetworkServiceEndpointRegistryClient that sets NetworkServiceNames for each incoming registration/find/unregistration.
func NewNetworkServiceEndpointRegistryClient(networkServices ...string) registry.NetworkServiceEndpointRegistryClient {
	return &setNetworkServiceNamesNSEClient{
		networkServices: networkServices,
	}
}
