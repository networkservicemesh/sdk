// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package setlogoption implements a chain element to set log options before full chain
package setlogoption

import (
	"context"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	nseRegistryClient = "NetworkServiceEndpointRegistryClient"
)

type setNSELogOptionClient struct {
	options map[string]string
}

func (s *setNSELogOptionClient) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	ctx = withFields(ctx, s.options, nseRegistryClient)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, endpoint, opts...)
}

func (s *setNSELogOptionClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ctx = withFields(ctx, s.options, nseRegistryClient)
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *setNSELogOptionClient) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = withFields(ctx, s.options, nseRegistryClient)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, endpoint, opts...)
}

// NewNetworkServiceEndpointRegistryClient creates new instance of NetworkServiceEndpointRegistryClient which sets the passed options
func NewNetworkServiceEndpointRegistryClient(options map[string]string) registry.NetworkServiceEndpointRegistryClient {
	return &setNSELogOptionClient{
		options: options,
	}
}
