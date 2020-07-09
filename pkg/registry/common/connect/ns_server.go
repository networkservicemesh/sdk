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

type nsCacheEntry struct {
	expiryTime time.Time
	client     registry.NetworkServiceRegistryClient
}

type connectNSServer struct {
	dialOptions       []grpc.DialOption
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient
	cache             map[string]*nsCacheEntry
	ctx               context.Context
	connectExpiration time.Duration
	executor          serialize.Executor
}

// NewNetworkServiceRegistryServer creates new connect NetworkServiceEndpointRegistryServer with specific chain context, registry client factory and options
// that allows connecting to other registries via passed clienturl.
func NewNetworkServiceRegistryServer(
	chainContext context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient,
	connectExpiration time.Duration,
	clientDialOptions ...grpc.DialOption) registry.NetworkServiceRegistryServer {
	return &connectNSServer{
		clientFactory:     clientFactory,
		cache:             map[string]*nsCacheEntry{},
		ctx:               chainContext,
		connectExpiration: connectExpiration,
		dialOptions:       clientDialOptions,
	}
}

func (c *connectNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	return c.connect(ctx).Register(ctx, ns)
}

func (c *connectNSServer) Find(q *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	return adapters.NetworkServiceClientToServer(c.connect(s.Context())).Find(q, s)
}

func (c *connectNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	return c.connect(ctx).Unregister(ctx, ns)
}

func (c *connectNSServer) cleanup() {
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

func (c *connectNSServer) connect(ctx context.Context) registry.NetworkServiceRegistryClient {
	defer c.executor.AsyncExec(c.cleanup)
	key := ""
	if url := clienturl.ClientURL(ctx); url != nil {
		key = url.String()
	}
	var result registry.NetworkServiceRegistryClient
	<-c.executor.AsyncExec(func() {
		if v, ok := c.cache[key]; ok && v != nil {
			v.expiryTime = time.Now().Add(c.connectExpiration)
			result = v.client
		}
	})
	if result != nil {
		return result
	}
	result = clienturl.NewNetworkServiceRegistryClient(ctx, c.clientFactory, c.dialOptions...)
	c.executor.AsyncExec(func() {
		c.cache[key] = &nsCacheEntry{client: result, expiryTime: time.Now().Add(c.connectExpiration)}
	})
	return result
}

var _ registry.NetworkServiceRegistryServer = (*connectNSServer)(nil)
