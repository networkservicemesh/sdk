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

// Package nextwrap provides adapters to wrap clients with no support to next, to support it.
package nextwrap

import (
	"context"
	"io"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type nextNetworkServiceEndpointWrappedClient struct {
	client registry.NetworkServiceEndpointRegistryClient
}

func (r *nextNetworkServiceEndpointWrappedClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	result, err := r.client.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, result, opts...)
}

func (r *nextNetworkServiceEndpointWrappedClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	client, err := r.client.Find(ctx, in, opts...)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if client != nil {
		return client, nil
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (r *nextNetworkServiceEndpointWrappedClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := r.client.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient wraps NetworkServiceEndpointRegistryClient to support next chain elements
func NewNetworkServiceEndpointRegistryClient(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	return &nextNetworkServiceEndpointWrappedClient{client: client}
}

var _ registry.NetworkServiceEndpointRegistryClient = &nextNetworkServiceEndpointWrappedClient{}
