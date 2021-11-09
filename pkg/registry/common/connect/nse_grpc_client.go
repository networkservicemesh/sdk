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

package connect

import (
	"context"
	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/registry"
)

type grpcNSEClient struct{}

func (c *grpcNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return registry.NewNetworkServiceEndpointRegistryClient(ccFromContext(ctx)).Register(ctx, nse, opts...)
}

func (c *grpcNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	for _, o := range opts {
		log.FromContext(ctx).WithField("FIND", "CLIENT").Debug(o)
	}
	return registry.NewNetworkServiceEndpointRegistryClient(ccFromContext(ctx)).Find(ctx, query, opts...)
}

func (c *grpcNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return registry.NewNetworkServiceEndpointRegistryClient(ccFromContext(ctx)).Unregister(ctx, nse, opts...)
}
