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

package connect_test

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cancelctx"
)

type monitorClient struct {
	ctx  context.Context
	cc   grpc.ClientConnInterface
	err  error
	lock sync.Mutex
	once sync.Once
}

func newMonitorClient(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
	return &monitorClient{
		ctx: ctx,
		cc:  cc,
	}
}

func (c *monitorClient) init() error {
	c.once.Do(func() {
		cancel := cancelctx.FromContext(c.ctx)
		if cancel == nil {
			c.err = errors.New("no cancel func")
			return
		}

		var stream grpc_health_v1.Health_WatchClient
		if stream, c.err = grpc_health_v1.NewHealthClient(c.cc).Watch(c.ctx, new(grpc_health_v1.HealthCheckRequest)); c.err != nil {
			return
		}

		go func() {
			defer cancel()
			for {
				if _, err := stream.Recv(); err != nil {
					c.lock.Lock()
					c.err = err
					c.lock.Unlock()
					return
				}
			}
		}()
	})

	c.lock.Lock()
	defer c.lock.Unlock()

	return c.err
}

func (c *monitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return next.Client(ctx).Request(ctx, request, opts...)
}

func (c *monitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	if err := c.init(); err != nil {
		return nil, err
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
