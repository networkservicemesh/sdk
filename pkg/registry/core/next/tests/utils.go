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

// Package tests contains tests for package 'next'
package tests

import (
	"context"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"

	"google.golang.org/grpc"
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

type testVisitNetworkServiceRegistryServer struct{}

func visitRegistryServer() registry.NetworkServiceRegistryServer {
	return &testVisitNetworkServiceRegistryServer{}
}

func (t *testVisitNetworkServiceRegistryServer) RegisterNSE(ctx context.Context, r *registry.NSERegistration) (*registry.NSERegistration, error) {
	return next.RegistryServer(visit(ctx)).RegisterNSE(ctx, r)
}

func (t *testVisitNetworkServiceRegistryServer) BulkRegisterNSE(registry.NetworkServiceRegistry_BulkRegisterNSEServer) error {
	return nil
}

func (t *testVisitNetworkServiceRegistryServer) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest) (*empty.Empty, error) {
	return next.RegistryServer(visit(ctx)).RemoveNSE(ctx, r)
}

type testVisitNetworkServiceRegistryClient struct{}

func visitRegistryClient() registry.NetworkServiceRegistryClient {
	return &testVisitNetworkServiceRegistryClient{}
}

func (t *testVisitNetworkServiceRegistryClient) RegisterNSE(ctx context.Context, r *registry.NSERegistration, _ ...grpc.CallOption) (*registry.NSERegistration, error) {
	return next.RegistryClient(visit(ctx)).RegisterNSE(ctx, r)
}

func (t *testVisitNetworkServiceRegistryClient) BulkRegisterNSE(ctx context.Context, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	return next.RegistryClient(visit(ctx)).BulkRegisterNSE(ctx)
}

func (t *testVisitNetworkServiceRegistryClient) RemoveNSE(ctx context.Context, r *registry.RemoveNSERequest, _ ...grpc.CallOption) (*empty.Empty, error) {
	return next.RegistryClient(visit(ctx)).RemoveNSE(ctx, r)
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
