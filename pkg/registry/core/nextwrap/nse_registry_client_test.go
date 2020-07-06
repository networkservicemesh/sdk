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

package nextwrap_test

import (
	"context"
	"io"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/nextwrap"
)

type emptyNSEClient struct{}

func (e *emptyNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return nil, nil
}

func (e *emptyNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return nil, nil
}

func (e *emptyNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

type nseFindClient struct {
	grpc.ClientStream
}

func (n *nseFindClient) Recv() (*registry.NetworkServiceEndpoint, error) {
	return nil, io.EOF
}

type nonEmptyNSEClient struct{}

func (n *nonEmptyNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return &registry.NetworkServiceEndpoint{}, nil
}

func (n *nonEmptyNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return &nseFindClient{}, nil
}

func (n *nonEmptyNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func TestWrapEmptyNetworkServiceEndpointClient(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		nextwrap.NewNetworkServiceEndpointRegistryClient(&emptyNSEClient{}),
		&nonEmptyNSEClient{})

	result, err := c.Register(context.Background(), nil)
	require.Nil(t, err)
	require.NotNil(t, result)

	client, err := c.Find(context.Background(), nil, nil)
	require.Nil(t, err)
	require.NotNil(t, client)
}

func TestWrapNonEmptyNetworkServiceEndpointClient(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		nextwrap.NewNetworkServiceEndpointRegistryClient(&nonEmptyNSEClient{}),
		&emptyNSEClient{})

	client, err := c.Find(context.Background(), nil, nil)
	require.Nil(t, err)
	require.NotNil(t, client)
}
