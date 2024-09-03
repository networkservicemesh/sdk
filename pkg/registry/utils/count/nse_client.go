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

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// countNSEClient is a client type for counting Register / Unregister / Find.
type countNSEClient struct {
	counter *CallCounter
}

// Register - increments registration call count.
func (c *countNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	atomic.AddInt32(&c.counter.totalRegisterCalls, 1)
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

// Unregister - increments un-registration call count.
func (c *countNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	atomic.AddInt32(&c.counter.totalUnregisterCalls, 1)
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}

// Find - increments find call count.
func (c *countNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	atomic.AddInt32(&c.counter.totalFindCalls, 1)
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

// NewNetworkServiceEndpointRegistryClient - creates a new chain element counting Register / Unregister / Find calls.
func NewNetworkServiceEndpointRegistryClient(counter *CallCounter) registry.NetworkServiceEndpointRegistryClient {
	return &countNSEClient{
		counter: counter,
	}
}
