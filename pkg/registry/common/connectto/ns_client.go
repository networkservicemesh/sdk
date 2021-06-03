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

package connectto

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
)

// NSClientFactory is a NS client chain supplier func type
type NSClientFactory = func() registry.NetworkServiceRegistryClient

type connectNSClient struct {
	ctx         context.Context
	client      registry.NetworkServiceRegistryClient
	connectTo   string
	dialOptions []grpc.DialOption

	cc   *grpc.ClientConn
	lock sync.RWMutex
}

// NewNetworkServiceRegistryClient returns a new NS registry client chain element connecting to the remote
//                                 NS registry server
func NewNetworkServiceRegistryClient(ctx context.Context, connectTo string, opts ...Option) registry.NetworkServiceRegistryClient {
	connectOpts := new(connectOptions)
	for _, opt := range opts {
		opt(connectOpts)
	}

	c := &connectNSClient{
		ctx: ctx,
		client: chain.NewNetworkServiceRegistryClient(
			append(
				connectOpts.nsAdditionalFunctionality,
				new(grpcNSClient),
			)...,
		),
		connectTo:   connectTo,
		dialOptions: append(append([]grpc.DialOption{}, connectOpts.dialOptions...), grpc.WithReturnConnectionError()),
	}

	go func() {
		<-ctx.Done()

		c.lock.Lock()
		defer c.lock.Unlock()

		if c.cc != nil {
			_ = c.cc.Close()
		}
	}()

	return c
}

func (c *connectNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	cc, err := c.getCC()
	if err != nil {
		return nil, err
	}
	return c.client.Register(withCC(ctx, cc), ns, opts...)
}

func (c *connectNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	cc, err := c.getCC()
	if err != nil {
		return nil, err
	}
	return c.client.Find(withCC(ctx, cc), query, opts...)
}

func (c *connectNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	cc, err := c.getCC()
	if err != nil {
		return nil, err
	}
	return c.client.Unregister(withCC(ctx, cc), ns, opts...)
}

func (c *connectNSClient) getCC() (*grpc.ClientConn, error) {
	c.lock.RLock()
	cc := c.cc
	c.lock.RUnlock()

	if cc != nil {
		return cc, nil
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	if c.cc != nil {
		return c.cc, nil
	}

	var err error
	if c.cc, err = grpc.DialContext(c.ctx, c.connectTo, c.dialOptions...); err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			c.lock.Lock()
			defer c.lock.Unlock()

			_ = c.cc.Close()
			c.cc = nil
		}()
		for c.cc.WaitForStateChange(c.ctx, c.cc.GetState()) {
			switch c.cc.GetState() {
			case connectivity.Connecting, connectivity.Idle, connectivity.Ready:
				continue
			default:
				return
			}
		}
	}()

	return c.cc, nil
}
