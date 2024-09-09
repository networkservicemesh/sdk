// Copyright (c) 2020-2021 Doc.ai, Inc.
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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	streamchannel "github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type echoNetworkServiceEndpointClient struct{}

func (t *echoNetworkServiceEndpointClient) Register(_ context.Context, in *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return in, nil
}

func (t *echoNetworkServiceEndpointClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, _ ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceEndpointResponse)
	go func() {
		ch <- &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: in.GetNetworkServiceEndpoint()}
		close(ch)
	}()
	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, ch), nil
}

func (t *echoNetworkServiceEndpointClient) Unregister(_ context.Context, _ *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

type echoNetworkServiceEndpointServer struct{}

func (e echoNetworkServiceEndpointServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return service, nil
}

func (e echoNetworkServiceEndpointServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	nseResp := &registry.NetworkServiceEndpointResponse{
		NetworkServiceEndpoint: query.GetNetworkServiceEndpoint(),
	}
	return server.Send(nseResp)
}

func (e echoNetworkServiceEndpointServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

type ignoreNSEFindServer struct {
	grpc.ServerStream
}

func (*ignoreNSEFindServer) Send(_ *registry.NetworkServiceEndpointResponse) error {
	return nil
}

func (*ignoreNSEFindServer) Context() context.Context {
	return context.Background()
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
		ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
		s := streamchannel.NewNetworkServiceEndpointFindServer(context.Background(), ch)
		err := server.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "test"}}, s)
		require.Nil(t, err)
		require.Equal(t, "test", (<-ch).GetNetworkServiceEndpoint().GetName())
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

type writeNSEServer struct{}

func (w *writeNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (w *writeNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	ctx := context.WithValue(server.Context(), testKey, true)
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, server))
}

func (w *writeNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

func TestNSEClientPassingContext(t *testing.T) {
	n := next.NewNetworkServiceEndpointRegistryClient(adapters.NetworkServiceEndpointServerToClient(&writeNSEServer{}), checkcontext.NewNSEClient(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value(testKey).(bool); !ok {
			t.Error("Context ignored")
		}
	}))

	_, err := n.Register(context.Background(), nil)
	require.NoError(t, err)

	_, err = n.Find(context.Background(), nil)
	require.NoError(t, err)

	_, err = n.Unregister(context.Background(), nil)
	require.NoError(t, err)
}

type writeNSEClient struct{}

func (s *writeNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (s *writeNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (s *writeNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func TestNSEPassingContext(t *testing.T) {
	n := next.NewNetworkServiceEndpointRegistryServer(adapters.NetworkServiceEndpointClientToServer(&writeNSEClient{}), checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value(testKey).(bool); !ok {
			t.Error("Context ignored")
		}
	}))

	_, err := n.Register(context.Background(), nil)
	require.NoError(t, err)

	err = n.Find(nil, streamcontext.NetworkServiceEndpointRegistryFindServer(context.Background(), nil))
	require.NoError(t, err)

	_, err = n.Unregister(context.Background(), nil)
	require.NoError(t, err)
}

func BenchmarkNSEServer_Register(b *testing.B) {
	server := adapters.NetworkServiceEndpointClientToServer(&echoNetworkServiceEndpointClient{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = server.Register(context.Background(), nil)
	}
}

func BenchmarkNSEServer_Find(b *testing.B) {
	server := adapters.NetworkServiceEndpointClientToServer(&echoNetworkServiceEndpointClient{})
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "test"},
		Watch:                  true,
	}
	findServer := &ignoreNSEFindServer{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.Find(query, findServer)
	}
}

func BenchmarkNSEClient_Find(b *testing.B) {
	client := adapters.NetworkServiceEndpointServerToClient(&echoNetworkServiceEndpointServer{})
	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "test"},
		Watch:                  true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ := client.Find(context.Background(), query)
		_, _ = c.Recv() // the only result
		_, _ = c.Recv() // EOF
	}
}
