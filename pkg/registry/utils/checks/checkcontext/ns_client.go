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

// Package checkcontext - provides registry chain elements for checking the context.Context passed on by the previous chain element
package checkcontext

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NewNSClient - returns NetworkServiceRegistryClient that checks the context passed in from the previous NSClient in the chain
//             t - *testing.T used for the check
//             check - function that checks the context.Context
func NewNSClient(t *testing.T, check func(*testing.T, context.Context)) registry.NetworkServiceRegistryClient {
	return &checkContextNSClient{
		T:     t,
		check: check,
	}
}

type checkContextNSClient struct {
	*testing.T
	check func(*testing.T, context.Context)
}

func (c *checkContextNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *checkContextNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (c *checkContextNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.check(c.T, ctx)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}
