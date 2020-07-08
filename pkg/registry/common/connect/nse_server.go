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
	"sync"
	"time"

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
	cacheLock         sync.Mutex
	ctx               context.Context
	once              sync.Once
	connectExpiration time.Duration
}

// NewNetworkServiceEndpointRegistryServer creates new connect NetworkServiceEndpointRegistryServer with specific chain context, registry client factory and options
// that allows connecting to other registries via passed clienturl.
func NewNetworkServiceEndpointRegistryServer(
	chainContext context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient,
	options ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &connectNSEServer{
		clientFactory:     clientFactory,
		cache:             map[string]*nseCacheEntry{},
		ctx:               chainContext,
		connectExpiration: time.Minute,
	}
	for _, o := range options {
		o.apply(r)
	}
	return r
}

func (c *connectNSEServer) Register(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	c.once.Do(c.init)
	return c.connect(ctx).Register(ctx, ns)
}

func (c *connectNSEServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	c.once.Do(c.init)
	return adapters.NetworkServiceEndpointClientToServer(c.connect(s.Context())).Find(q, s)
}

func (c *connectNSEServer) Unregister(ctx context.Context, ns *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	c.once.Do(c.init)
	return c.connect(ctx).Unregister(ctx, ns)
}

func (c *connectNSEServer) init() {
	go func() {
		defer func() {
			c.cacheLock.Lock()
			c.cache = nil
			c.cacheLock.Unlock()
		}()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-time.After(c.connectExpiration):
				c.cacheLock.Lock()
				var deleteList []string
				for k, v := range c.cache {
					if time.Until(v.expiryTime) <= 0 {
						deleteList = append(deleteList, k)
					}
				}
				for _, item := range deleteList {
					delete(c.cache, item)
				}
				c.cacheLock.Unlock()
			}
		}
	}()
}

func (c *connectNSEServer) connect(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	key := ""
	if url := clienturl.ClientURL(ctx); url != nil {
		key = url.String()
	}
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if v, ok := c.cache[key]; ok && v != nil {
		v.expiryTime = time.Now().Add(c.connectExpiration)
		return v.client
	}
	r := clienturl.NewNetworkServiceEndpointRegistryClient(ctx, c.clientFactory, c.dialOptions...)
	c.cache[key] = &nseCacheEntry{client: r, expiryTime: time.Now().Add(c.connectExpiration)}
	return r
}

func (c *connectNSEServer) setExpirationDuration(d time.Duration) {
	c.connectExpiration = d
}

func (c *connectNSEServer) setClientDialOptions(opts []grpc.DialOption) {
	c.dialOptions = opts
}

var _ registry.NetworkServiceEndpointRegistryServer = (*connectNSEServer)(nil)
