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

type echoNetworkServiceEndpointClient struct{}

func (t *echoNetworkServiceEndpointClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return in, nil
}

func (t *echoNetworkServiceEndpointClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, _ ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEndpoint)
	go func() {
		ch <- in.NetworkServiceEndpoint
		close(ch)
	}()
	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, ch), nil
}

func (t *echoNetworkServiceEndpointClient) Unregister(_ context.Context, _ *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

func TestNetworkServiceEndpointClientToServer_Register(t *testing.T) {
	expected := &registry.NetworkServiceEndpoint{
		Name: "echo",
	}
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceEndpointClientToServerFewTimes(i, &echoNetworkServiceEndpointClient{})
		actual, err := server.Register(context.Background(), expected)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}
}

func TestNetworkServiceEndpointClientToServer_Unregister(t *testing.T) {
	expected := &registry.NetworkServiceEndpoint{
		Name: "echo",
	}
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceEndpointClientToServerFewTimes(i, &echoNetworkServiceEndpointClient{})
		_, err := server.Unregister(context.Background(), expected)
		require.NoError(t, err)
	}
}

func TestNetworkServiceEndpointFind(t *testing.T) {
	for i := 1; i < adaptCountPerTest; i++ {
		server := adaptNetworkServiceEndpointClientToServerFewTimes(i, &echoNetworkServiceEndpointClient{})
		ch := make(chan *registry.NetworkServiceEndpoint, 1)
		s := streamchannel.NewNetworkServiceEndpointFindServer(context.Background(), ch)
		err := server.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "test"}}, s)
		require.Nil(t, err)
		require.Equal(t, "test", (<-ch).Name)
		close(ch)
	}
}

func adaptNetworkServiceEndpointClientToServerFewTimes(n int, client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryServer {
	if n == 0 {
		panic("n should be more or equal 1")
	}
	s := adapters.NetworkServiceEndpointClientToServer(client)

	for i := 1; i < n; i++ {
		client = adapters.NetworkServiceEndpointServerToClient(s)
		s = adapters.NetworkServiceEndpointClientToServer(client)
	}

	return s
}
