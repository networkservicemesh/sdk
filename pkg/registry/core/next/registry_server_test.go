// Copyright (c) 2020 Doc.ai and/or its affiliates.
//
// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package next_test contains tests for package 'next'
package next_test

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"testing"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

type contextKeyType string

const (
	visitKey contextKeyType = "visitKey"
)

func visit(ctx context.Context) context.Context {
	if v, ok := ctx.Value(visitKey).(*int); ok {
		*v++
		return ctx
	}
	val := 0
	return context.WithValue(ctx, visitKey, &val)
}

func visitValue(ctx context.Context) int {
	if v, ok := ctx.Value(visitKey).(*int); ok {
		return *v
	}
	return 0
}

type testVisitRegistryServer struct{}

func visitRegistryServer() registry.NetworkServiceRegistryServer {
	return &testVisitRegistryServer{}
}

func (t *testVisitRegistryServer) RegisterNSE(ctx context.Context, r *registry.NSERegistration) (*registry.NSERegistration, error) {
	return next.RegistryServer(visit(ctx)).RegisterNSE(ctx, r)
}

func (t *testVisitRegistryServer) BulkRegisterNSE(registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return nil
}

func (t *testVisitRegistryServer) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.RegistryServer(visit(ctx)).RemoveNSE(ctx, r)
}

type testEmptyRegistryServer struct{}

func (t *testEmptyRegistryServer) RegisterNSE(context.Context, *registry.NSERegistration) (*registry.NSERegistration, error) {
	return nil, nil
}

func (t *testEmptyRegistryServer) BulkRegisterNSE(registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return nil
}

func (t *testEmptyRegistryServer) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest) (*empty.Empty, error) {
	return nil, nil
}

func emptyRegistryServer() registry.NetworkServiceRegistryServer {
	return &testEmptyRegistryServer{}
}

func TestNewRegistryServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewRegistryServer().RegisterNSE(context.Context(nil), nil)
	})
}

func TestServerBranches(t *testing.T) {
	servers := [][]registry.NetworkServiceRegistryServer{
		{visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), visitRegistryServer()},
		{emptyRegistryServer(), visitRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), emptyRegistryServer(), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), emptyRegistryServer()},
		{adapters.NewRegistryClientToServer(visitRegistryClient())},
		{adapters.NewRegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient())},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(emptyRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.NewRegistryClientToServer(visitRegistryClient()), emptyRegistryServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 1, 2, 3, 3, 1, 2}
	for i, sample := range servers {
		s := next.NewRegistryServer(sample...)

		ctx := visit(context.Background())
		_, _ = s.RegisterNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_, _ = s.RemoveNSE(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}
