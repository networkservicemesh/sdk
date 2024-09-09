// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package count

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// countNSClient is a client type for counting Register / Unregister / Find.
type countNSClient struct {
	counter *CallCounter
}

// Register - increments registration call count.
func (c *countNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	atomic.AddInt32(&c.counter.totalRegisterCalls, 1)
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

// Unregister - increments un-registration call count.
func (c *countNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	atomic.AddInt32(&c.counter.totalUnregisterCalls, 1)
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// Find - increments find call count.
func (c *countNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	atomic.AddInt32(&c.counter.totalFindCalls, 1)
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

// NewNetworkServiceRegistryClient - creates a new chain element counting Register / Unregister / Find calls.
func NewNetworkServiceRegistryClient(counter *CallCounter) registry.NetworkServiceRegistryClient {
	return &countNSClient{
		counter: counter,
	}
}
