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

// Package clients provides adapters to wrap clients with no support to next, to support it.
package clients

import (
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"google.golang.org/grpc"
)

type nextWrappedClient struct {
	client registry.NetworkServiceRegistryClient
}

func (r *nextWrappedClient) Register(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*registry.NetworkServiceEntry, error) {
	result, err := r.client.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, result, opts...)
}

func (r *nextWrappedClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	_, err := r.client.Find(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (r *nextWrappedClient) Unregister(ctx context.Context, in *registry.NetworkServiceEntry, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := r.client.Unregister(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func NewNetworkServiceRegistryClient(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &nextWrappedClient{client: client}
}

var _ registry.NetworkServiceRegistryClient = &nextWrappedClient{}
