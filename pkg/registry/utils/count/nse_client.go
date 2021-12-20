// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NSEClient is a client type for counting Register / Unregister / Find
type NSEClient struct {
	totalRegisterCalls, totalUnregisterCalls, totalFindCalls int32
	registers, unregisters, finds                            map[string]int32
	mu                                                       sync.Mutex
}

// Register - increments registration call count
func (c *NSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalRegisterCalls, 1)
	if c.registers == nil {
		c.registers = make(map[string]int32)
	}
	c.registers[nse.GetName()]++

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

// Unregister - increments un-registration call count
func (c *NSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalUnregisterCalls, 1)
	if c.unregisters == nil {
		c.unregisters = make(map[string]int32)
	}
	c.unregisters[nse.GetName()]++

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}

// Find - increments find call count
func (c *NSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	atomic.AddInt32(&c.totalFindCalls, 1)
	if c.unregisters == nil {
		c.unregisters = make(map[string]int32)
	}
	c.unregisters[query.String()]++

	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

// Registers returns Register call count
func (c *NSEClient) Registers() int {
	return int(atomic.LoadInt32(&c.totalRegisterCalls))
}

// Unregisters returns Unregister count
func (c *NSEClient) Unregisters() int {
	return int(atomic.LoadInt32(&c.totalUnregisterCalls))
}

// Finds returns Find count
func (c *NSEClient) Finds() int {
	return int(atomic.LoadInt32(&c.totalFindCalls))
}
