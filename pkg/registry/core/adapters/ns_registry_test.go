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

type echoNetworkServiceClient struct{}

func (t *echoNetworkServiceClient) Register(_ context.Context, in *registry.NetworkService, _ ...grpc.CallOption) (*registry.NetworkService, error) {
	return in, nil
}

func (t *echoNetworkServiceClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ch := make(chan *registry.NetworkServiceResponse)
	go func() {
		ch <- &registry.NetworkServiceResponse{NetworkService: in.GetNetworkService()}
		close(ch)
	}()
	return streamchannel.NewNetworkServiceFindClient(ctx, ch), nil
}

func (t *echoNetworkServiceClient) Unregister(_ context.Context, _ *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

type echoNetworkServiceServer struct{}

func (e echoNetworkServiceServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	return service, nil
}

func (e echoNetworkServiceServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	nsResp := &registry.NetworkServiceResponse{
		NetworkService: query.GetNetworkService(),
	}
	return server.Send(nsResp)
}

func (e echoNetworkServiceServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	return &empty.Empty{}, nil
}

type ignoreNSFindServer struct {
	grpc.ServerStream
}

func (*ignoreNSFindServer) Send(_ *registry.NetworkServiceResponse) error {
	return nil
}

func (*ignoreNSFindServer) Context() context.Context {
	return context.Background()
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
		ch := make(chan *registry.NetworkServiceResponse, 1)
		s := streamchannel.NewNetworkServiceFindServer(context.Background(), ch)
		err := server.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: "test"}}, s)
		require.Nil(t, err)
		require.Equal(t, "test", (<-ch).GetNetworkService().GetName())
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

type contextKeyType string

const testKey contextKeyType = "TestContext"

type writeNSServer struct{}

func (w *writeNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (w *writeNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	ctx := context.WithValue(server.Context(), testKey, true)
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, streamcontext.NetworkServiceRegistryFindServer(ctx, server))
}

func (w *writeNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}

func TestNSClientPassingContext(t *testing.T) {
	n := next.NewNetworkServiceRegistryClient(adapters.NetworkServiceServerToClient(&writeNSServer{}), checkcontext.NewNSClient(t, func(t *testing.T, ctx context.Context) {
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

type writeNSClient struct{}

func (w *writeNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (w *writeNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (w *writeNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx = context.WithValue(ctx, testKey, true)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func TestNSServerPassingContext(t *testing.T) {
	n := next.NewNetworkServiceRegistryServer(adapters.NetworkServiceClientToServer(&writeNSClient{}), checkcontext.NewNSServer(t, func(t *testing.T, ctx context.Context) {
		if _, ok := ctx.Value(testKey).(bool); !ok {
			t.Error("Context ignored")
		}
	}))

	_, err := n.Register(context.Background(), nil)
	require.NoError(t, err)

	err = n.Find(nil, streamcontext.NetworkServiceRegistryFindServer(context.Background(), nil))
	require.NoError(t, err)

	_, err = n.Unregister(context.Background(), nil)
	require.NoError(t, err)
}

func BenchmarkNSServer_Register(b *testing.B) {
	server := adapters.NetworkServiceClientToServer(&echoNetworkServiceClient{})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = server.Register(context.Background(), nil)
	}
}

func BenchmarkNSServer_Find(b *testing.B) {
	server := adapters.NetworkServiceClientToServer(&echoNetworkServiceClient{})
	query := &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{Name: "test"},
		Watch:          true,
	}
	findServer := &ignoreNSFindServer{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = server.Find(query, findServer)
	}
}

func BenchmarkNSClient_Find(b *testing.B) {
	client := adapters.NetworkServiceServerToClient(&echoNetworkServiceServer{})
	query := &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{Name: "test"},
		Watch:          true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c, _ := client.Find(context.Background(), query)
		_, _ = c.Recv() // the only result
		_, _ = c.Recv() // EOF
	}
}
