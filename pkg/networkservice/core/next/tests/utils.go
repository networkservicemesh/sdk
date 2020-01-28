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

	"github.com/golang/protobuf/ptypes/empty"
	
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
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

type testVisitNetworkServiceServer struct{}

func visitServer() networkservice.NetworkServiceServer {
	return &testVisitNetworkServiceServer{}
}

func (t *testVisitNetworkServiceServer) Request(ctx context.Context, r *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return next.Server(visit(ctx)).Request(ctx, r)
}

func (t *testVisitNetworkServiceServer) Close(ctx context.Context, r *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(visit(ctx)).Close(ctx, r)
}

type testVisitNetworkServiceClient struct{}

func visitClient() networkservice.NetworkServiceClient {
	return &testVisitNetworkServiceClient{}
}

func (t *testVisitNetworkServiceClient) Request(ctx context.Context, r *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return next.Client(visit(ctx)).Request(ctx, r, opts...)
}

func (t *testVisitNetworkServiceClient) Close(ctx context.Context, r *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.Client(visit(ctx)).Close(ctx, r, opts...)
}

type testEmptyClient struct{}

func (t testEmptyClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return nil, nil
}

func (t testEmptyClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

func emptyClient() networkservice.NetworkServiceClient {
	return &testEmptyClient{}
}

type testEmptyServer struct{}

func (t testEmptyServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return nil, nil
}

func (t testEmptyServer) Close(ctx context.Context, in *networkservice.Connection) (*empty.Empty, error) {
	return nil, nil
}

func emptyServer() networkservice.NetworkServiceServer {
	return &testEmptyServer{}
}
