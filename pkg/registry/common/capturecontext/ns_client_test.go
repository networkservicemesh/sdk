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
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/capturecontext"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type contextKeyType string

const writerNSKey contextKeyType = "writerNSKey"

type checkContextNSClient struct{}

func (c *checkContextNSClient) Register(ctx context.Context, _ *registry.NetworkService, _ ...grpc.CallOption) (*registry.NetworkService, error) {
	return nil, checkContext(ctx, writerNSKey)
}

func (c *checkContextNSClient) Find(ctx context.Context, _ *registry.NetworkServiceQuery, _ ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return nil, checkContext(ctx, writerNSKey)
}

func (c *checkContextNSClient) Unregister(ctx context.Context, _ *registry.NetworkService, _ ...grpc.CallOption) (*empty.Empty, error) {
	return nil, checkContext(ctx, writerNSKey)
}

func checkContext(ctx context.Context, key contextKeyType) error {
	if capturecontext.CapturedContext(ctx).Value(key) == true {
		return nil
	}
	return errors.New("Context not found")
}

type writeNSClient struct{}

func (w *writeNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func (w *writeNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func (w *writeNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(capturecontext.WithCapturedContext(ctx), in, opts...)
}

func TestNSClientContextStorage(t *testing.T) {
	chain := next.NewNetworkServiceRegistryClient(&writeNSClient{}, capturecontext.NewNetworkServiceRegistryClient(), &checkContextNSClient{})

	ctx := context.WithValue(context.Background(), writerNSKey, true)
	_, err := chain.Register(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Find(ctx, nil)
	require.NoError(t, err)

	_, err = chain.Unregister(ctx, nil)
	require.NoError(t, err)
}
