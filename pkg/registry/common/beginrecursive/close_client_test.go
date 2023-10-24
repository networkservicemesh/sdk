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

package beginrecursive_test

import (
	"context"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	mark = "mark"
)

func TestCloseClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		&markClient{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	resp, err := client.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: id,
	})
	assert.NotNil(t, t, resp)
	assert.NoError(t, err)
	assert.Equal(t, mark, resp.GetNetworkServiceLabels()[mark].Labels[mark])
	resp = resp.Clone()
	delete(resp.GetNetworkServiceLabels()[mark].Labels, mark)
	assert.Empty(t, resp.GetNetworkServiceLabels()[mark].Labels)
	_, err = client.Unregister(ctx, resp)
	assert.NoError(t, err)
}

type markClient struct {
	t *testing.T
}

func (m *markClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if in.GetNetworkServiceLabels() == nil {
		in.NetworkServiceLabels = make(map[string]*registry.NetworkServiceLabels)
	}

	in.GetNetworkServiceLabels()[mark] = &registry.NetworkServiceLabels{
		Labels: map[string]string{
			mark: mark,
		},
	}

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (m *markClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (m *markClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	assert.NotNil(m.t, in.GetNetworkServiceLabels())
	assert.Equal(m.t, mark, in.GetNetworkServiceLabels()[mark].Labels[mark])
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

func TestDoubleCloseClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		&doubleCloseClient{t: t},
	)
	id := "1"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, err := client.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: id,
	})
	assert.NotNil(t, t, conn)
	assert.NoError(t, err)
	conn = conn.Clone()
	_, err = client.Unregister(ctx, conn)
	assert.NoError(t, err)
	_, err = client.Unregister(ctx, conn)
	assert.NoError(t, err)
}

type doubleCloseClient struct {
	t *testing.T
	sync.Once
}

func (s *doubleCloseClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (s *doubleCloseClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (s *doubleCloseClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	count := 1
	s.Do(func() {
		count++
	})
	assert.Equal(s.t, 2, count, "Close has been called more than once")
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}
