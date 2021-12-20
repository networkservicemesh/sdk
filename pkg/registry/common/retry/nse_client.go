// Copyright (c) 2021 Cisco and/or its affiliates.
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

package retry

import (
	"context"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type retryNSEClient struct {
	interval   time.Duration
	tryTimeout time.Duration
}

// NewNetworkServiceEndpointRegistryClient - returns a retry chain element
func NewNetworkServiceEndpointRegistryClient(opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	client := &retryNSEClient{
		interval:   time.Millisecond * 200,
		tryTimeout: time.Second * 15,
	}

	for _, opt := range opts {
		opt.apply(client)
	}

	return client
}

func (r *retryNSEClient) setInterval(interval time.Duration) {
	r.interval = interval
}

func (r *retryNSEClient) setTryTimeout(tryTimeout time.Duration) {
	r.tryTimeout = tryTimeout
}

func (r *retryNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	logger := log.FromContext(ctx).WithField("retryNSEClient", "Register")
	c := clock.FromContext(ctx)

	for ctx.Err() == nil {
		registerCtx, cancel := c.WithTimeout(ctx, r.tryTimeout)
		resp, err := next.NetworkServiceEndpointRegistryClient(registerCtx).Register(registerCtx, nse, opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.After(r.interval):
				continue
			}
		}

		return resp, err
	}

	return nil, ctx.Err()
}

func (r *retryNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	logger := log.FromContext(ctx).WithField("retryNSEClient", "Find")
	c := clock.FromContext(ctx)

	for ctx.Err() == nil {
		stream, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())
			<-c.After(r.interval)
			continue
		}

		return stream, err
	}

	return nil, ctx.Err()
}

func (r *retryNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	logger := log.FromContext(ctx).WithField("retryNSEClient", "Unregister")
	c := clock.FromContext(ctx)

	for ctx.Err() == nil {
		closeCtx, cancel := c.WithTimeout(ctx, r.tryTimeout)
		resp, err := next.NetworkServiceEndpointRegistryClient(closeCtx).Unregister(closeCtx, in, opts...)
		cancel()

		if err != nil {
			logger.Errorf("try attempt has failed: %v", err.Error())

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-c.After(r.interval):
				continue
			}
		}

		return resp, err
	}

	return nil, ctx.Err()
}
