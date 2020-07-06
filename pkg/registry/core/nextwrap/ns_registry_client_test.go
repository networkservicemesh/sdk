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

type emptyNSClient struct{}

func (e *emptyNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return nil, nil
}

func (e *emptyNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return nil, nil
}

func (e *emptyNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

type nonEmptyNSClient struct{}

func (n *nonEmptyNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return &registry.NetworkService{}, nil
}

type nsFindClient struct {
	grpc.ClientStream
}

func (n *nsFindClient) Recv() (*registry.NetworkService, error) {
	return nil, io.EOF
}

func (n *nonEmptyNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return &nsFindClient{}, nil
}

func (n *nonEmptyNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func TestWrapEmptyNetworkServiceClient(t *testing.T) {
	c := next.NewNetworkServiceRegistryClient(
		nextwrap.NewNetworkServiceRegistryClient(&emptyNSClient{}),
		&nonEmptyNSClient{})

	result, err := c.Register(context.Background(), nil)
	require.Nil(t, err)
	require.NotNil(t, result)

	client, err := c.Find(context.Background(), nil, nil)
	require.Nil(t, err)
	require.NotNil(t, client)
}

func TestWrapNonEmptyNetworkServiceClient(t *testing.T) {
	c := next.NewNetworkServiceRegistryClient(
		nextwrap.NewNetworkServiceRegistryClient(&nonEmptyNSClient{}),
		&emptyNSClient{},
	)

	client, err := c.Find(context.Background(), nil, nil)
	require.Nil(t, err)
	require.NotNil(t, client)
}
