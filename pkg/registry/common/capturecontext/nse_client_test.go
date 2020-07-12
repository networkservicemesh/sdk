// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package capturecontext_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/capturecontext"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const writerNSEKey contextKeyType = "writerNSEKey"

type checkContextNSEClient struct{}

func (c *checkContextNSEClient) Register(ctx context.Context, _ *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return nil, checkContext(ctx, writerNSEKey)
}

func (c *checkContextNSEClient) Find(ctx context.Context, _ *registry.NetworkServiceEndpointQuery, _ ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return nil, checkContext(ctx, writerNSEKey)
}

func (c *checkContextNSEClient) Unregister(ctx context.Context, _ *registry.NetworkServiceEndpoint, _ ...grpc.CallOption) (*empty.Empty, error) {
	return nil, checkContext(ctx, writerNSEKey)
}

type writeNSEClient struct{}

func (w *writeNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func (w *writeNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func (w *writeNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func TestNSEClientContextStorage(t *testing.T) {
	chain := next.NewNetworkServiceEndpointRegistryClient(&writeNSEClient{}, capturecontext.NewNetworkServiceEndpointRegistryClient(), &checkContextNSEClient{})

	ctx := context.WithValue(context.Background(), writerNSEKey, true)
	_, err := chain.Register(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Find(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Unregister(ctx, nil)
	require.NoError(t, err)
}
