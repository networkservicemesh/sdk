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

package connect

import (
	"context"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

type nseCacheEntry struct {
	expiryTime time.Time
	client     registry.NetworkServiceEndpointRegistryClient
}

type connectNSEServer struct {
	dialOptions       []grpc.DialOption
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient
	cache             map[string]*nseCacheEntry
	connectExpiration time.Duration
	executor          serialize.Executor
}

// NewNetworkServiceEndpointRegistryServer creates new connect NetworkServiceEndpointEndpointRegistryServer with specific chain context, registry client factory and options
// that allows connecting to other registries via passed clienturl.
func NewNetworkServiceEndpointRegistryServer(
	chainContext context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient,
	connectExpiration time.Duration,
	clientDialOptions ...grpc.DialOption) registry.NetworkServiceEndpointRegistryServer {
	return &connectNSEServer{
		clientFactory:     clientFactory,
		cache:             map[string]*nseCacheEntry{},
		connectExpiration: connectExpiration,
		dialOptions:       clientDialOptions,
	}
}

func (c *connectNSEServer) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return c.connect(ctx).Register(ctx, ns)
}

func (c *connectNSEServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return adapters.NetworkServiceEndpointClientToServer(c.connect(s.Context())).Find(q, s)
}

func (c *connectNSEServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return c.connect(ctx).Unregister(ctx, ns)
}

func (c *connectNSEServer) cleanup() {
	var deleteList []string
	for k, v := range c.cache {
		if time.Until(v.expiryTime) <= 0 {
			deleteList = append(deleteList, k)
		}
	}
	for _, item := range deleteList {
		delete(c.cache, item)
	}
}

func (c *connectNSEServer) connect(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	defer c.executor.AsyncExec(c.cleanup)
	key := ""
	if url := clienturl.ClientURL(ctx); url != nil {
		key = url.String()
	}
	var result registry.NetworkServiceEndpointRegistryClient
	<-c.executor.AsyncExec(func() {
		if v, ok := c.cache[key]; ok && v != nil {
			v.expiryTime = time.Now().Add(c.connectExpiration)
			result = v.client
		}
	})
	if result != nil {
		return result
	}
	result = clienturl.NewNetworkServiceEndpointRegistryClient(ctx, c.clientFactory, c.dialOptions...)
	c.executor.AsyncExec(func() {
		c.cache[key] = &nseCacheEntry{client: result, expiryTime: time.Now().Add(c.connectExpiration)}
	})
	return result
}

var _ registry.NetworkServiceEndpointRegistryServer = (*connectNSEServer)(nil)
