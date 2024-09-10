// Copyright (c) 2024 Cisco and/or its affiliates.
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

package limit

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"google.golang.org/grpc"
)

type limitNSEClient struct {
	cfg *limitConfig
}

func (n *limitNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleNSEClient", "Register")

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing conneciton...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (n *limitNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	}

	logger := log.FromContext(ctx).WithField("throttleNSEClient", "Find")
	doneCh := make(chan struct{})
	defer close(doneCh)

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing conneciton...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)

	if err == nil {
		go func() {
			select {
			case <-time.After(n.cfg.dialLimit):
				logger.Warn("Reached dial limit, closing conneciton...")
				_ = closer.Close()
			case <-resp.Context().Done():
				return
			}
		}()
	}

	return resp, err
}

func (n *limitNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleNSEClient", "Unregister")

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing conneciton...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient - returns a new null client that does nothing but call next.NetworkServiceEndpointRegistryClient(ctx).
func NewNetworkServiceEndpointRegistryClient(opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	cfg := &limitConfig{
		dialLimit: time.Minute,
	}

	for _, opt := range opts[:] {
		opt(cfg)
	}

	return &limitNSEClient{
		cfg: cfg,
	}
}
