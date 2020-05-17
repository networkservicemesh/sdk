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
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

type testVisitRegistryClient struct{}

func visitRegistryClient() registry.NetworkServiceRegistryClient {
	return &testVisitRegistryClient{}
}

func (t *testVisitRegistryClient) RegisterNSE(ctx context.Context, r *registry.NSERegistration, _ ...grpc.CallOption) (*registry.NSERegistration, error) {
	return next.NetworkServiceRegistryClient(visit(ctx)).RegisterNSE(ctx, r)
}

func (t *testVisitRegistryClient) BulkRegisterNSE(ctx context.Context, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	return next.NetworkServiceRegistryClient(visit(ctx)).BulkRegisterNSE(ctx)
}

func (t *testVisitRegistryClient) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest, _ ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(visit(ctx)).RemoveNSE(ctx, r)
}

type testEmptyRegistryClient struct{}

func (t *testEmptyRegistryClient) RegisterNSE(context.Context, *registry.NSERegistration, ...grpc.CallOption) (*registry.NSERegistration, error) {
	return nil, nil
}

func (t *testEmptyRegistryClient) BulkRegisterNSE(context.Context, ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	return nil, nil
}

func (t *testEmptyRegistryClient) RemoveNSE(context.Context, *registry.RemoveNSERequest, ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func emptyRegistryClient() registry.NetworkServiceRegistryClient {
	return &testEmptyRegistryClient{}
}

func TestNewRegistryClientShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceRegistryClient().RegisterNSE(context.Context(nil), nil)
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
		{adapters.NewRegistryServerToClient(visitRegistryServer())},
		{adapters.NewRegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer())},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(emptyRegistryServer()), visitRegistryClient()},
		{visitRegistryClient(), adapters.NewRegistryServerToClient(visitRegistryServer()), emptyRegistryClient()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2}
	for i, sample := range samples {
		msg := fmt.Sprintf("sample index: %v", i)
		s := next.NewNetworkServiceRegistryClient(sample...)

		ctx := visit(context.Background())
		_, _ = s.RegisterNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.RemoveNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)

		ctx = visit(context.Background())
		_, _ = s.BulkRegisterNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), msg)
	}
}
