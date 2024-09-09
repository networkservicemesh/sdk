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

package next_test

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/stretchr/testify/assert"
)

type testVisitNSERegistryServer struct{}

func (t *testVisitNSERegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(visit(ctx), request)
}

func (t *testVisitNSERegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	s = streamcontext.NetworkServiceEndpointRegistryFindServer(visit(s.Context()), s)
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (t *testVisitNSERegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(visit(ctx), request)
}

func visitNSERegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &testVisitNSERegistryServer{}
}

type testEmptyNSERegistryServer struct{}

func (t *testEmptyNSERegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return nil, nil
}

func (t *testEmptyNSERegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return io.EOF
}

func (t *testEmptyNSERegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return nil, nil
}

func emptyNSERegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &testEmptyNSERegistryServer{}
}

func TestNewNetworkServiceEndpointRegistryServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceEndpointRegistryServer().Register(context.Background(), nil)
		_, _ = next.NewWrappedNetworkServiceEndpointRegistryServer(func(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
			return server
		}).Register(context.Background(), nil)
	})
}

func TestNSEServerBranches(t *testing.T) {
	servers := [][]registry.NetworkServiceEndpointRegistryServer{
		{visitNSERegistryServer()},
		{visitNSERegistryServer(), visitNSERegistryServer()},
		{visitNSERegistryServer(), visitNSERegistryServer(), visitNSERegistryServer()},
		{emptyNSERegistryServer(), visitNSERegistryServer(), visitNSERegistryServer()},
		{visitNSERegistryServer(), emptyNSERegistryServer(), visitNSERegistryServer()},
		{visitNSERegistryServer(), visitNSERegistryServer(), emptyNSERegistryServer()},
		{visitNSERegistryServer(), adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient())},
		{adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient()), visitNSERegistryServer()},
		{visitNSERegistryServer(), adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient()), visitNSERegistryServer()},
		{visitNSERegistryServer(), visitNSERegistryServer(), adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient())},
		{visitNSERegistryServer(), adapters.NetworkServiceEndpointClientToServer(emptyNSERegistryClient()), visitNSERegistryServer()},
		{visitNSERegistryServer(), adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient()), emptyNSERegistryServer()},
		{adapters.NetworkServiceEndpointClientToServer(visitNSERegistryClient()), adapters.NetworkServiceEndpointClientToServer(emptyNSERegistryClient()), visitNSERegistryServer()},
		{next.NewNetworkServiceEndpointRegistryServer(), next.NewNetworkServiceEndpointRegistryServer(visitNSERegistryServer(), next.NewNetworkServiceEndpointRegistryServer()), visitNSERegistryServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2, 1, 2}
	for i, sample := range servers {
		s := next.NewNetworkServiceEndpointRegistryServer(sample...)

		ctx := visit(context.Background())
		_, _ = s.Register(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_, _ = s.Unregister(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_ = s.Find(nil, streamcontext.NetworkServiceEndpointRegistryFindServer(ctx, nil))

		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}

type testVisitNSERegistryClient struct{}

func (t *testVisitNSERegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(visit(ctx), in)
}

func (t *testVisitNSERegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(visit(ctx), in, opts...)
}

func (t *testVisitNSERegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(visit(ctx), in)
}

func visitNSERegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return &testVisitNSERegistryClient{}
}

type testEmptyNSERegistryClient struct{}

func (t *testEmptyNSERegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return nil, nil
}

type testEmptyNSERegistryFindClient struct {
	grpc.ClientStream
	ctx context.Context
}

func (t *testEmptyNSERegistryFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	return nil, io.EOF
}

func (t *testEmptyNSERegistryFindClient) Context() context.Context {
	return t.ctx
}

func (t *testEmptyNSERegistryClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return &testEmptyNSERegistryFindClient{ctx: ctx}, nil
}

func (t *testEmptyNSERegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func emptyNSERegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return &testEmptyNSERegistryClient{}
}

func TestNewNetworkServiceEndpointRegistryClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceEndpointRegistryClient().Register(context.Background(), nil)
		_, _ = next.NewWrappedNetworkServiceEndpointRegistryClient(func(client registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
			return client
		}).Register(context.Background(), nil)
	})
}

func TestNetworkServiceEndpointRegistryClientBranches(t *testing.T) {
	samples := [][]registry.NetworkServiceEndpointRegistryClient{
		{visitNSERegistryClient()},
		{visitNSERegistryClient(), visitNSERegistryClient()},
		{visitNSERegistryClient(), visitNSERegistryClient(), visitNSERegistryClient()},
		{emptyNSERegistryClient(), visitNSERegistryClient(), visitNSERegistryClient()},
		{visitNSERegistryClient(), emptyNSERegistryClient(), visitNSERegistryClient()},
		{visitNSERegistryClient(), visitNSERegistryClient(), emptyNSERegistryClient()},
		{visitNSERegistryClient(), adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer())},
		{adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer()), visitNSERegistryClient()},
		{visitNSERegistryClient(), adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer()), visitNSERegistryClient()},
		{visitNSERegistryClient(), visitNSERegistryClient(), adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer())},
		{visitNSERegistryClient(), adapters.NetworkServiceEndpointServerToClient(emptyNSERegistryServer()), visitNSERegistryClient()},
		{visitNSERegistryClient(), adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer()), emptyNSERegistryClient()},
		{adapters.NetworkServiceEndpointServerToClient(visitNSERegistryServer()), adapters.NetworkServiceEndpointServerToClient(emptyNSERegistryServer()), visitNSERegistryClient()},
		{next.NewNetworkServiceEndpointRegistryClient(), next.NewNetworkServiceEndpointRegistryClient(visitNSERegistryClient(), next.NewNetworkServiceEndpointRegistryClient()), visitNSERegistryClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2, 1, 2}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		s := next.NewNetworkServiceEndpointRegistryClient(sample...)

		ctx := visit(context.Background())
		_, _ = s.Register(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.Unregister(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.Find(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)
	}
}

func TestDataRaceNetworkServiceEndpointServer(t *testing.T) {
	s := next.NewNetworkServiceEndpointRegistryServer(emptyNSERegistryServer())
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.Register(context.Background(), nil)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.Find(nil, streamcontext.NetworkServiceEndpointRegistryFindServer(context.Background(), nil))
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.Unregister(context.Background(), nil)
		}()
	}
	wg.Wait()
}

func TestDataRaceNetworkServiceEndpointClient(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(emptyNSERegistryClient())
	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Register(context.Background(), nil)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Find(context.Background(), nil)
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Unregister(context.Background(), nil)
		}()
	}
	wg.Wait()
}
