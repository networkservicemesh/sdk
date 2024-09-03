// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package cleanup

import (
	"context"
	"sync/atomic"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type cleanupClient struct {
	chainCtx context.Context

	ccClose     bool
	doneCh      chan struct{}
	activeConns int32
	executor    serialize.Executor
}

// NewClient - returns a cleanup client chain element.
func NewClient(ctx context.Context, opts ...Option) networkservice.NetworkServiceClient {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	c := &cleanupClient{
		chainCtx: ctx,
		ccClose:  o.ccClose,
		doneCh:   o.doneCh,
	}
	go func() {
		<-c.chainCtx.Done()
		if atomic.LoadInt32(&c.activeConns) == 0 && c.doneCh != nil {
			c.executor.AsyncExec(func() {
				select {
				case <-c.doneCh:
				default:
					close(c.doneCh)
				}
			})
		}
	}()
	return c
}

func (c *cleanupClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	// Update active connections counter. Needed for a cleanup done notification.
	atomic.AddInt32(&c.activeConns, 1)
	if cancel, ok := loadAndDeleteCancel(ctx); ok {
		cancel()
	}

	cancelCtx, cancel := context.WithCancel(context.Background())
	storeCancel(ctx, cancel)

	factory := begin.FromContext(ctx)
	go func() {
		select {
		case <-c.chainCtx.Done():
			// Add to metadata if we want to delete clientconn
			if c.ccClose {
				storeCC(ctx)
			}

			<-factory.Close(begin.CancelContext(cancelCtx))
			atomic.AddInt32(&c.activeConns, -1)

			if atomic.LoadInt32(&c.activeConns) == 0 && c.doneCh != nil {
				c.executor.AsyncExec(func() {
					select {
					case <-c.doneCh:
					default:
						close(c.doneCh)
					}
				})
			}
		case <-cancelCtx.Done():
			atomic.AddInt32(&c.activeConns, -1)
		}
	}()
	return conn, nil
}

func (c *cleanupClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cancel, ok := loadAndDeleteCancel(ctx); ok {
		if _, ok := loadAndDeleteCC(ctx); ok {
			if cc, ok := clientconn.Load(ctx); ok {
				if closable, ok := cc.(interface{ Close() error }); ok {
					_ = closable.Close()
				}
				clientconn.Delete(ctx)
			}
		}
		cancel()
	}
	return next.Client(ctx).Close(ctx, conn, opts...)
}
