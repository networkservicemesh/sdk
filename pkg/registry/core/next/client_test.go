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

package next_test

import (
	"context"
	"fmt"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamcontext"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/stretchr/testify/assert"
)

type testVisitRegistryClient struct {
	names []string
}

func (t testVisitRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(visit(ctx), in)
}

type testlServiceRegistryFindClient struct {
	grpc.ClientStream
}

func (t testlServiceRegistryFindClient) Context() context.Context {
	return context.Background()
}

func (t testlServiceRegistryFindClient) Recv() (*registry.NetworkServiceEntry, error) {
	return nil, nil
}

func (t testVisitRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return streamcontext.NetworkServiceRegistryFindClient(visit(ctx), &testlServiceRegistryFindClient{}), nil
}

func (t testVisitRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(visit(ctx), in)
}

func visitRegistryClient() registry.NetworkServiceRegistryClient {
	return &testVisitRegistryClient{}
}

type testEmptyRegistryClient struct {
}

func (t *testEmptyRegistryClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	return nil, nil
}

func (t *testEmptyRegistryClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return nil, nil
}

func (t *testEmptyRegistryClient) Unregister(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func emptyRegistryClient() registry.NetworkServiceRegistryClient {
	return &testEmptyRegistryClient{}
}

func TestNewRegistryClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceRegistryClient().Register(context.Context(nil), nil)
	})
}

func TestRegistryClientBranches(t *testing.T) {
	samples := [][]registry.NetworkServiceRegistryClient{
		{visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), visitRegistryClient()},
		{emptyRegistryClient(), visitRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), emptyRegistryClient(), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), emptyRegistryClient()},
		{visitRegistryClient(), adapters.RegistryServerToClient(visitRegistryServer())},
		{adapters.RegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.RegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), adapters.RegistryServerToClient(visitRegistryServer())},
		{visitRegistryClient(), adapters.RegistryServerToClient(emptyRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.RegistryServerToClient(visitRegistryServer()), emptyRegistryClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		s := next.NewNetworkServiceRegistryClient(sample...)

		ctx := visit(context.Background())
		_, _ = s.Register(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.Unregister(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		//ctx = visit(context.Background())
		//_, _ = s.Find(ctx, nil)
		//assert.Equal(t, expects[i], visitValue(ctx), msg)
	}
}
