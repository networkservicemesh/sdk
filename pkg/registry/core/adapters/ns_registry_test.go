// Copyright (c) 2020 Doc.ai, Inc.
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

package adapters_test

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type echoNetworkServiceClient struct{}

func (t *echoNetworkServiceClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return in, nil
}

func (t *echoNetworkServiceClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkService)
	go func() {
		ch <- in.NetworkService
		close(ch)
	}()
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}

func (t *echoNetworkServiceClient) Unregister(_ context.Context, _ *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func TestNetworkServiceClientToServer_Register(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceClientToServerFewTimes(i, &echoNetworkServiceClient{})
		actual, err := server.Register(context.Background(), expected)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func TestNetworkServiceClientToServer_Unregister(t *testing.T) {
	expected := &registry.NetworkService{
		Name: "echo",
	}
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceClientToServerFewTimes(i, &echoNetworkServiceClient{})
		_, err := server.Unregister(context.Background(), expected)
		require.NoError(t, err)
	}
}

func TestNetworkServiceFind(t *testing.T) {
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceClientToServerFewTimes(i, &echoNetworkServiceClient{})
		ch := make(chan *registry.NetworkService, 1)
		s := streamchannel.NewNetworkServiceFindServer(context.Background(), ch)
		err := server.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: "test"}}, s)
		require.Nil(t, err)
		require.Equal(t, "test", (<-ch).Name)
		close(ch)
	}
}

func adaptNetworkServiceClientToServerFewTimes(n int, client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryServer {
	if n == 0 {
		panic("n should be more or equal 1")
	}
	s := adapters.NetworkServiceClientToServer(client)

	for i := 1; i < n; i++ {
		client = adapters.NetworkServiceServerToClient(s)
		s = adapters.NetworkServiceClientToServer(client)
	}

	return s
}
