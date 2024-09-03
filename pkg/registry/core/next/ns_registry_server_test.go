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

type testVisitNSRegistryServer struct{}

func (t *testVisitNSRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryServer(ctx).Register(visit(ctx), request)
}

func (t *testVisitNSRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	s = streamcontext.NetworkServiceRegistryFindServer(visit(s.Context()), s)
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (t *testVisitNSRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).Unregister(visit(ctx), request)
}

func visitNSRegistryServer() registry.NetworkServiceRegistryServer {
	return &testVisitNSRegistryServer{}
}

type testEmptyNSRegistryServer struct{}

func (t *testEmptyNSRegistryServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	return nil, nil
}

func (t *testEmptyNSRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	return io.EOF
}

func (t *testEmptyNSRegistryServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	return nil, nil
}

func emptyNSRegistryServer() registry.NetworkServiceRegistryServer {
	return &testEmptyNSRegistryServer{}
}

func TestNewNetworkServiceRegistryServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceRegistryServer().Register(context.Background(), nil)
		_, _ = next.NewWrappedNetworkServiceRegistryServer(func(server registry.NetworkServiceRegistryServer) registry.NetworkServiceRegistryServer {
			return server
		}).Register(context.Background(), nil)
	})
}

func TestNSServerBranches(t *testing.T) {
	servers := [][]registry.NetworkServiceRegistryServer{
		{visitNSRegistryServer()},
		{visitNSRegistryServer(), visitNSRegistryServer()},
		{visitNSRegistryServer(), visitNSRegistryServer(), visitNSRegistryServer()},
		{emptyNSRegistryServer(), visitNSRegistryServer(), visitNSRegistryServer()},
		{visitNSRegistryServer(), emptyNSRegistryServer(), visitNSRegistryServer()},
		{visitNSRegistryServer(), visitNSRegistryServer(), emptyNSRegistryServer()},
		{visitNSRegistryServer(), adapters.NetworkServiceClientToServer(visitNSRegistryClient())},
		{adapters.NetworkServiceClientToServer(visitNSRegistryClient()), visitNSRegistryServer()},
		{visitNSRegistryServer(), adapters.NetworkServiceClientToServer(visitNSRegistryClient()), visitNSRegistryServer()},
		{visitNSRegistryServer(), visitNSRegistryServer(), adapters.NetworkServiceClientToServer(visitNSRegistryClient())},
		{visitNSRegistryServer(), adapters.NetworkServiceClientToServer(emptyNSRegistryClient()), visitNSRegistryServer()},
		{visitNSRegistryServer(), adapters.NetworkServiceClientToServer(visitNSRegistryClient()), emptyNSRegistryServer()},
		{adapters.NetworkServiceClientToServer(visitNSRegistryClient()), adapters.NetworkServiceClientToServer(emptyNSRegistryClient()), visitNSRegistryServer()},
		{next.NewNetworkServiceRegistryServer(), next.NewNetworkServiceRegistryServer(visitNSRegistryServer(), next.NewNetworkServiceRegistryServer()), visitNSRegistryServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2, 1, 2}
	for i, sample := range servers {
		s := next.NewNetworkServiceRegistryServer(sample...)

		ctx := visit(context.Background())
		_, _ = s.Register(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_, _ = s.Unregister(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_ = s.Find(nil, streamcontext.NetworkServiceRegistryFindServer(ctx, nil))

		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}

type testVisitNSRegistryClient struct{}

func (t *testVisitNSRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(visit(ctx), in)
}

func (t *testVisitNSRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(visit(ctx), in, opts...)
}

func (t *testVisitNSRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(visit(ctx), in)
}

func visitNSRegistryClient() registry.NetworkServiceRegistryClient {
	return &testVisitNSRegistryClient{}
}

type testEmptyNSRegistryClient struct{}

func (t *testEmptyNSRegistryClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return nil, nil
}

type testEmptyNSRegistryFindClient struct {
	grpc.ClientStream
	ctx context.Context
}

func (t *testEmptyNSRegistryFindClient) Recv() (*registry.NetworkServiceResponse, error) {
	return nil, io.EOF
}

func (t *testEmptyNSRegistryFindClient) Context() context.Context {
	return t.ctx
}

func (t *testEmptyNSRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return &testEmptyNSRegistryFindClient{ctx: ctx}, nil
}

func (t *testEmptyNSRegistryClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func emptyNSRegistryClient() registry.NetworkServiceRegistryClient {
	return &testEmptyNSRegistryClient{}
}

func TestNewNetworkServiceRegistryClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceRegistryClient().Register(context.Background(), nil)
		_, _ = next.NewWrappedNetworkServiceRegistryClient(func(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
			return client
		}).Register(context.Background(), nil)
	})
}

func TestDataRaceNetworkServiceServer(t *testing.T) {
	s := next.NewNetworkServiceRegistryServer(emptyNSRegistryServer())
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
			_ = s.Find(nil, streamcontext.NetworkServiceRegistryFindServer(context.Background(), nil))
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = s.Unregister(context.Background(), nil)
		}()
	}
	wg.Wait()
}

func TestDataRaceNetworkServiceClient(t *testing.T) {
	c := next.NewNetworkServiceRegistryClient(emptyNSRegistryClient())
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

func TestNetworkServiceRegistryClientBranches(t *testing.T) {
	samples := [][]registry.NetworkServiceRegistryClient{
		{visitNSRegistryClient()},
		{visitNSRegistryClient(), visitNSRegistryClient()},
		{visitNSRegistryClient(), visitNSRegistryClient(), visitNSRegistryClient()},
		{emptyNSRegistryClient(), visitNSRegistryClient(), visitNSRegistryClient()},
		{visitNSRegistryClient(), emptyNSRegistryClient(), visitNSRegistryClient()},
		{visitNSRegistryClient(), visitNSRegistryClient(), emptyNSRegistryClient()},
		{visitNSRegistryClient(), adapters.NetworkServiceServerToClient(visitNSRegistryServer())},
		{adapters.NetworkServiceServerToClient(visitNSRegistryServer()), visitNSRegistryClient()},
		{visitNSRegistryClient(), adapters.NetworkServiceServerToClient(visitNSRegistryServer()), visitNSRegistryClient()},
		{visitNSRegistryClient(), visitNSRegistryClient(), adapters.NetworkServiceServerToClient(visitNSRegistryServer())},
		{visitNSRegistryClient(), adapters.NetworkServiceServerToClient(emptyNSRegistryServer()), visitNSRegistryClient()},
		{visitNSRegistryClient(), adapters.NetworkServiceServerToClient(visitNSRegistryServer()), emptyNSRegistryClient()},
		{adapters.NetworkServiceServerToClient(visitNSRegistryServer()), adapters.NetworkServiceServerToClient(emptyNSRegistryServer()), visitNSRegistryClient()},
		{next.NewNetworkServiceRegistryClient(), next.NewNetworkServiceRegistryClient(visitNSRegistryClient(), next.NewNetworkServiceRegistryClient()), visitNSRegistryClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2, 1, 2}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		s := next.NewNetworkServiceRegistryClient(sample...)

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
