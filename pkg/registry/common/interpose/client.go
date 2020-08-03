// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

// Package interpose provides NetworkServiceRegistryServer that registers local Endpoints
// and adds them to Map
package interpose

import (
	"context"
	"strings"

	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type interposeClient struct {
}

func (u *interposeClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if !strings.HasPrefix(in.Name, interposeNSEName) {
		in.Name = interposeNSEName + in.Name
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (u *interposeClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (u *interposeClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if !strings.HasPrefix(in.Name, interposeNSEName) {
		in.Name = interposeNSEName + in.Name
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient - creates a Client that will replace any passed endpoint with CrossConnect NSE name for proper registration
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return &interposeClient{}
}

var _ registry.NetworkServiceEndpointRegistryClient = (*interposeClient)(nil)
