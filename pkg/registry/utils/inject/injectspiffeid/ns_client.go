// Copyright (c) 2022 Cisco and/or its affiliates.
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

package injectspiffeid

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type injectSpiffeIDNSClient struct {
	cert []byte
}

// NewNetworkServiceRegistryClient returns a client chain element putting spiffeID to context on Register and Unregister
func NewNetworkServiceRegistryClient(spiffeID string) registry.NetworkServiceRegistryClient {
	return &injectSpiffeIDNSClient{
		cert: generateCert(spiffeID),
	}
}

func (c *injectSpiffeIDNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	ctx, err := withPeer(ctx, c.cert)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
}

func (c *injectSpiffeIDNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *injectSpiffeIDNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	ctx, err := withPeer(ctx, c.cert)
	if err != nil {
		return nil, err
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}
