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
	"errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

type registryServerToClient struct {
	client registry.NetworkServiceRegistryClient
}

// NewNextRegistryClient - returns a new registry.NetworkServiceRegistryClient that is a wrapper around client
func NewNextRegistryClient(client registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	return &registryServerToClient{client: client}
}

func (r *registryServerToClient) RegisterNSE(ctx context.Context, registration *registry.NSERegistration, opts ...grpc.CallOption) (*registry.NSERegistration, error) {
	result, err := r.client.RegisterNSE(ctx, registration)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).RegisterNSE(ctx, result, opts...)
}

// BulkRegisterNSE - register in bulk
func (r *registryServerToClient) BulkRegisterNSE(ctx context.Context, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_BulkRegisterNSEClient, error) {
	// TODO: will be removed
	return nil, errors.New("not supported, and will be removed")
}

func (r *registryServerToClient) RemoveNSE(ctx context.Context, request *registry.RemoveNSERequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	_, err := r.client.RemoveNSE(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).RemoveNSE(ctx, request, opts...)
}

// Implementation check
var _ registry.NetworkServiceRegistryClient = &registryServerToClient{}
