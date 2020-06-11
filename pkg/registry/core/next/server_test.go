// Copyright (c) 2020 Cisco Systems, Inc.
//
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

// Package next_test contains tests for package 'next'
package next_test

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/stretchr/testify/assert"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"testing"
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

func (t *testVisitRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEntry) (*registry.NetworkServiceEntry, error) {
	return next.NetworkServiceRegistryServer(ctx).Register(visit(ctx), request)
}

func (t *testVisitRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (t *testVisitRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEntry) (*empty.Empty, error) {
	return next.NetworkServiceRegistryServer(ctx).Unregister(visit(ctx), request)
}

func visitRegistryServer() registry.NetworkServiceRegistryServer {
	return &testVisitRegistryServer{}
}

type testEmptyRegistryServer struct{}

func (t *testEmptyRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEntry) (*registry.NetworkServiceEntry, error) {
	return nil, nil
}

func (t *testEmptyRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	return nil
}

func (t *testEmptyRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEntry) (*empty.Empty, error) {
	return nil, nil
}

func emptyRegistryServer() registry.NetworkServiceRegistryServer {
	return &testEmptyRegistryServer{}
}

func TestNewRegistryServerShouldNotPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		_, _ = next.NewNetworkServiceRegistryServer().Register(context.Context(nil), nil)
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
		{visitRegistryServer(), adapters.RegistryClientToServer(visitRegistryClient())},
		{adapters.RegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.RegistryClientToServer(visitRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), visitRegistryServer(), adapters.RegistryClientToServer(visitRegistryClient())},
		{visitRegistryServer(), adapters.RegistryClientToServer(emptyRegistryClient()), visitRegistryServer()},
		{visitRegistryServer(), adapters.RegistryClientToServer(visitRegistryClient()), emptyRegistryServer()},
	}
	expects := []int{1, 2, 3, 0, 1, 2, 2, 2, 3, 3, 1, 2}
	for i, sample := range servers {
		s := next.NewNetworkServiceRegistryServer(sample...)

		ctx := visit(context.Background())
		_, _ = s.Register(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		ctx = visit(context.Background())
		_, _ = s.Unregister(ctx, nil)
		assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))

		//ctx = visit(context.Background())
		//_ = s.Find(nil, nil)
		//
		//assert.Equal(t, expects[i], visitValue(ctx), fmt.Sprintf("sample index: %v", i))
	}
}
