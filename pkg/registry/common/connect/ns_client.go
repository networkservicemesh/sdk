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

package connect

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

type connectNSClient struct {
	ctx           context.Context
	clientFactory NSClientFactory
	dialOptions   []grpc.DialOption
	initOnce      sync.Once
	dialErr       error
	client        registry.NetworkServiceRegistryClient
}

func (c *connectNSClient) init() error {
	c.initOnce.Do(func() {
		ctx, cancel := context.WithCancel(c.ctx)
		c.ctx = ctx

		clientURL := clienturlctx.ClientURL(c.ctx)
		if clientURL == nil {
			c.dialErr = errors.New("cannot dial nil clienturl.ClientURL(ctx)")
			cancel()
			return
		}

		dialOptions := append(append([]grpc.DialOption{}, c.dialOptions...), grpc.WithReturnConnectionError())

		var cc *grpc.ClientConn
		cc, c.dialErr = grpc.DialContext(ctx, grpcutils.URLToTarget(clientURL), dialOptions...)
		if c.dialErr != nil {
			cancel()
			return
		}

		c.client = c.clientFactory(c.ctx, cc)

		go func() {
			defer func() {
				cancel()
				_ = cc.Close()
			}()

			stream, err := grpc_health_v1.NewHealthClient(cc).Watch(c.ctx, new(grpc_health_v1.HealthCheckRequest))
			for ; err == nil; _, err = stream.Recv() {
			}
		}()
	})

	return c.dialErr
}

func (c *connectNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.Register(ctx, ns, opts...)
}

func (c *connectNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.Find(ctx, query, opts...)
}

func (c *connectNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return c.client.Unregister(ctx, ns, opts...)
}
