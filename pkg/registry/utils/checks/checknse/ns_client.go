// Copyright (c) 2022 Cisco and/or its affiliates.
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

package checknse

import (
	"context"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type checkNSClient struct {
	*testing.T
	check func(*testing.T, context.Context, *registry.NetworkService)
}

// NewNetworkServiceRegistryClient - returns NetworkServiceRegistryClient that checks the NS passed in from the previous NSClient in the chain
//
//	t - *testing.T used for the check
//	check - function that checks the *registry.NetworkService
func NewNetworkServiceRegistryClient(t *testing.T, check func(*testing.T, context.Context, *registry.NetworkService)) registry.NetworkServiceRegistryClient {
	return &checkNSClient{
		T:     t,
		check: check,
	}
}

func (c *checkNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	c.check(c.T, ctx, ns)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
}

func (c *checkNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *checkNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.check(c.T, ctx, ns)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}
