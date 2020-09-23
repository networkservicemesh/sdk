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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

type nseCacheEntry struct {
	expirationTimer *time.Timer
	client          registry.NetworkServiceEndpointRegistryClient
}

type connectNSEServer struct {
	dialOptions       []grpc.DialOption
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient
	cache             nseClientMap
	connectExpiration time.Duration
	ctx               context.Context
}

// NewNetworkServiceEndpointRegistryServer creates new connect NetworkServiceEndpointEndpointRegistryServer with specific chain context, registry client factory and options
// that allows connecting to other registries via passed clienturl.
// ctx - a context for all lifecycle
func NewNetworkServiceEndpointRegistryServer(ctx context.Context,
	clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient,
	options ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &connectNSEServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		connectExpiration: defaultConnectExpiration,
	}
	for _, o := range options {
		o.apply(r)
	}
	return r
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

func (c *connectNSEServer) connect(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	key := ""
	if url := clienturlctx.ClientURL(ctx); url != nil {
		key = url.String()
	}
	if v, ok := c.cache.Load(key); v != nil && ok {
		if v.expirationTimer.Stop() {
			v.expirationTimer.Reset(c.connectExpiration)
		}
		return v.client
	}
	ctx = extend.WithValuesFromContext(c.ctx, ctx)
	client := clienturl.NewNetworkServiceEndpointRegistryClient(ctx, c.clientFactory, c.dialOptions...)
	cached, _ := c.cache.LoadOrStore(key, &nseCacheEntry{
		expirationTimer: time.AfterFunc(c.connectExpiration, func() {
			c.cache.Delete(key)
		}),
		client: client,
	})
	return cached.client
}

func (c *connectNSEServer) setExpirationDuration(d time.Duration) {
	c.connectExpiration = d
}

func (c *connectNSEServer) setClientDialOptions(opts []grpc.DialOption) {
	c.dialOptions = opts
}

var _ registry.NetworkServiceEndpointRegistryServer = (*connectNSEServer)(nil)
