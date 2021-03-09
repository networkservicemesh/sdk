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

// Package localbypass implements a chain element to set NSMgr URL to endpoints on registration and set back endpoints
// URLs on find
package localbypass

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type localBypassNSEClient struct {
	url string
}

// NewNetworkServiceEndpointRegistryClient creates new instance of NetworkServiceEndpointRegistryClient which
// checks endpoints URLs on find
func NewNetworkServiceEndpointRegistryClient(u string) registry.NetworkServiceEndpointRegistryClient {
	return &localBypassNSEClient{
		url: u,
	}
}

func (rc *localBypassNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (rc *localBypassNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	client, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	if err != nil {
		return client, err
	}
	return &localBypassNSEFindClient{
		url: rc.url,
		NetworkServiceEndpointRegistry_FindClient: client,
	}, nil
}

func (rc *localBypassNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}
