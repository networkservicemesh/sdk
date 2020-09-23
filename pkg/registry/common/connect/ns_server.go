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

type nsCacheEntry struct {
	expirationTimer *time.Timer
	client          registry.NetworkServiceRegistryClient
}

type connectNSServer struct {
	dialOptions       []grpc.DialOption
	clientFactory     func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient
	cache             nsClientMap
	connectExpiration time.Duration
	ctx               context.Context
}

// NewNetworkServiceRegistryServer creates new connect NetworkServiceEndpointRegistryServer with specific chain context, registry client factory and options
// that allows connecting to other registries via passed clienturl.
func NewNetworkServiceRegistryServer(ctx context.Context, clientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient, options ...Option) registry.NetworkServiceRegistryServer {
	r := &connectNSServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		connectExpiration: defaultConnectExpiration,
	}
	for _, o := range options {
		o.apply(r)
	}
	return r
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

func (c *connectNSServer) connect(ctx context.Context) registry.NetworkServiceRegistryClient {
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

	// Use new context with longer lifetime to use with client
	ctx = extend.WithValuesFromContext(c.ctx, ctx)
	client := clienturl.NewNetworkServiceRegistryClient(ctx, c.clientFactory, c.dialOptions...)
	cached, _ := c.cache.LoadOrStore(key, &nsCacheEntry{
		expirationTimer: time.AfterFunc(c.connectExpiration, func() {
			c.cache.Delete(key)
		}),
		client: client,
	})
	return cached.client
}

func (c *connectNSServer) setExpirationDuration(d time.Duration) {
	c.connectExpiration = d
}

func (c *connectNSServer) setClientDialOptions(opts []grpc.DialOption) {
	c.dialOptions = opts
}

var _ registry.NetworkServiceRegistryServer = (*connectNSServer)(nil)
