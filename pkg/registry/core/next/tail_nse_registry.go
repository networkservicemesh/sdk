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

package next

import (
	"context"
	"io"

	"google.golang.org/grpc"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

// tailNetworkServiceEndpointRegistryServer is a simple implementation of registry.NetworkServiceEndpointRegistryServer that is called at the end
// of a chain to ensure that we never call a method on a nil object
type tailNetworkServiceEndpointRegistryServer struct{}

func (t *tailNetworkServiceEndpointRegistryServer) Register(_ context.Context, r *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return r, nil
}

func (t *tailNetworkServiceEndpointRegistryServer) Find(_ *registry.NetworkServiceEndpointQuery, _ registry.NetworkServiceEndpointRegistry_FindServer) error {
	return nil
}

func (t *tailNetworkServiceEndpointRegistryServer) Unregister(context.Context, *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryServer = &tailNetworkServiceEndpointRegistryServer{}

// tailNetworkServiceEndpointRegistryClient is a simple implementation of registry.NetworkServiceEndpointRegistryClient that is called at the end
// of a chain to ensure that we never call a method on a nil object
type tailNetworkServiceEndpointRegistryClient struct{}

func (t *tailNetworkServiceEndpointRegistryClient) Register(_ context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return in, nil
}

type tailNetworkServiceEndpointRegistryFindClient struct {
	grpc.ClientStream
	ctx context.Context
}

func (t *tailNetworkServiceEndpointRegistryFindClient) Context() context.Context {
	return t.ctx
}

func (t *tailNetworkServiceEndpointRegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	return nil, io.EOF
}

func (t *tailNetworkServiceEndpointRegistryClient) Find(ctx context.Context, _ *registry.NetworkServiceEndpointQuery, _ ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	return &tailNetworkServiceEndpointRegistryFindClient{ctx: ctx}, nil
}

func (t *tailNetworkServiceEndpointRegistryClient) Unregister(_ context.Context, _ *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

var _ registry.NetworkServiceEndpointRegistryClient = &tailNetworkServiceEndpointRegistryClient{}
