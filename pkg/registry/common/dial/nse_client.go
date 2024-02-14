// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

// Package dial will dial up a grpc.ClientConnInterface if a client *url.URL is provided in the ctx, retrievable by
// clienturlctx.ClientURL(ctx) and put the resulting grpc.ClientConnInterface into the ctx using clientconn.Store(..)
// where it can be retrieved by other chain elements using clientconn.Load(...)
package dial

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type dialNSEClient struct {
	chainCtx    context.Context
	dialOptions []grpc.DialOption
	dialTimeout time.Duration
}

func (c *dialNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	var di = newDialer(c.chainCtx, c.dialTimeout, c.dialOptions...)
	clientconn.Store(ctx, di)
	err := di.Dial(ctx, clienturlctx.ClientURL(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		clientconn.Delete(ctx)
		_ = di.Close()
	}()
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}
func (c *dialNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	var di = newDialer(c.chainCtx, c.dialTimeout, c.dialOptions...)
	clientconn.Store(ctx, di)
	err := di.Dial(ctx, clienturlctx.ClientURL(ctx))
	if err != nil {
		return nil, err
	}
	defer func() {
		clientconn.Delete(ctx)
		_ = di.Close()
	}()
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

type dialNSEFindClient struct {
	registry.NetworkServiceEndpointRegistry_FindClient
	cleanupFn func()
}

func (c *dialNSEFindClient) Recv() (*registry.NetworkServiceEndpointResponse, error) {
	resp, err := c.NetworkServiceEndpointRegistry_FindClient.Recv()
	if err != nil {
		c.cleanupFn()
	}
	return resp, err
}

func (c *dialNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	}

	di := newDialer(c.chainCtx, c.dialTimeout, c.dialOptions...)

	err := di.Dial(ctx, clientURL)
	if err != nil {
		log.FromContext(ctx).Errorf("can not dial to %v, err %v. Deleting clientconn...", grpcutils.URLToTarget(clientURL), err)
		return nil, err
	}

	clientconn.Store(ctx, di)

	cleanupFn := func() {
		clientconn.Delete(ctx)
		_ = di.Close()
	}

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	if err != nil {
		cleanupFn()
		return nil, err
	}

	go func() {
		<-resp.Context().Done()
		cleanupFn()
	}()

	return &dialNSEFindClient{
		NetworkServiceEndpointRegistry_FindClient: resp,
		cleanupFn: cleanupFn,
	}, nil
}

// NewNetworkServiceEndpointRegistryClient - returns a new null client that does nothing but call next.NetworkServiceEndpointRegistryClient(ctx).
func NewNetworkServiceEndpointRegistryClient(chainCtx context.Context, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	o := &option{
		dialTimeout: time.Millisecond * 300,
	}
	for _, opt := range opts {
		opt(o)
	}
	return &dialNSEClient{
		chainCtx:    chainCtx,
		dialOptions: o.dialOptions,
		dialTimeout: o.dialTimeout,
	}
}
