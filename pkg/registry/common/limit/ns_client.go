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

// Package limit provides a chain element that can set limits for the RPC calls.
package limit

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type limitNSClient struct {
	cfg *limitConfig
}

func (n *limitNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleNSClient", "Register")

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing connection...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (n *limitNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
	}

	logger := log.FromContext(ctx).WithField("throttleNSClient", "Find")

	doneCh := make(chan struct{})
	defer close(doneCh)

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing connection...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()

	resp, err := next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
	if err == nil {
		go func() {
			select {
			case <-time.After(n.cfg.dialLimit):
				logger.Warn("Reached dial limit, closing connection...")
				_ = closer.Close()
			case <-resp.Context().Done():
				return
			}
		}()
	}
	return resp, err
}

func (n *limitNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	cc, ok := clientconn.Load(ctx)
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
	}

	closer, ok := cc.(interface{ Close() error })
	if !ok {
		return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := log.FromContext(ctx).WithField("throttleNSClient", "Unregister")

	go func() {
		select {
		case <-time.After(n.cfg.dialLimit):
			logger.Warn("Reached dial limit, closing connection...")
			_ = closer.Close()
		case <-doneCh:
			return
		}
	}()

	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - returns a new null client that does nothing but call next.NetworkServiceRegistryClient(ctx).
func NewNetworkServiceRegistryClient(opts ...Option) registry.NetworkServiceRegistryClient {
	cfg := &limitConfig{
		dialLimit: time.Minute,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return &limitNSClient{
		cfg: cfg,
	}
}
