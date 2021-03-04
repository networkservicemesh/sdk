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

package querycache_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	name           = "nse"
	url1           = "tcp://1.1.1.1"
	url2           = "tcp://2.2.2.2"
	expirationTime = 100 * time.Millisecond
)

func testNSEQuery(nseName string) *registry.NetworkServiceEndpointQuery {
	return &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: nseName,
		},
	}
}

func Test_QueryCacheClient_ShouldCacheNSEs(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	failureClient := new(failureNSEClient)
	c := next.NewNetworkServiceEndpointRegistryClient(
		querycache.NewClient(ctx, time.Minute),
		failureClient,
		adapters.NetworkServiceEndpointServerToClient(mem),
	)

	reg, err := mem.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: name,
		Url:  url1,
	})
	require.NoError(t, err)

	// Goroutines should be cleaned up on NSE unregister
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Find from memory
	stream, err := c.Find(ctx, testNSEQuery(""))
	require.NoError(t, err)

	nse, err := stream.Recv()
	require.NoError(t, err)

	require.Equal(t, name, nse.Name)
	require.Equal(t, url1, nse.Url)

	// 2. Find from cache
	atomic.StoreInt32(&failureClient.shouldFail, 1)

	require.Eventually(t, func() bool {
		if stream, err = c.Find(ctx, testNSEQuery(name)); err != nil {
			return false
		}
		if nse, err = stream.Recv(); err != nil {
			return false
		}
		return name == nse.Name && url1 == nse.Url
	}, 100*time.Millisecond, time.Millisecond)

	// 3. Update NSE in memory
	reg.Url = url2

	reg, err = mem.Register(ctx, reg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		if stream, err = c.Find(ctx, testNSEQuery(name)); err != nil {
			return false
		}
		if nse, err = stream.Recv(); err != nil {
			return false
		}
		return name == nse.Name && url2 == nse.Url
	}, 100*time.Millisecond, time.Millisecond)

	// 4. Delete NSE from memory
	_, err = mem.Unregister(ctx, reg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err = c.Find(ctx, testNSEQuery(name))
		return err != nil
	}, 100*time.Millisecond, time.Millisecond)
}

func Test_QueryCacheClient_ShouldCleanUpOnTimeout(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	failureClient := new(failureNSEClient)
	c := next.NewNetworkServiceEndpointRegistryClient(
		querycache.NewClient(ctx, expirationTime),
		failureClient,
		adapters.NetworkServiceEndpointServerToClient(mem),
	)

	_, err := mem.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: name,
	})
	require.NoError(t, err)

	// Goroutines should be cleaned up on cache entry expiration
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Find from memory
	stream, err := c.Find(ctx, testNSEQuery(""))
	require.NoError(t, err)

	_, err = stream.Recv()
	require.NoError(t, err)

	// 2. Find from cache
	atomic.StoreInt32(&failureClient.shouldFail, 1)

	require.Eventually(t, func() bool {
		if stream, err = c.Find(ctx, testNSEQuery(name)); err == nil {
			_, err = stream.Recv()
		}
		return err == nil
	}, 100*time.Millisecond, time.Millisecond)

	// 3. Keep finding from cache to prevent expiration
	for start := time.Now(); time.Since(start) > 2*expirationTime; time.Sleep(expirationTime / 10) {
		stream, err = c.Find(ctx, testNSEQuery(""))
		require.NoError(t, err)

		_, err = stream.Recv()
		require.NoError(t, err)
	}

	// 4. Wait for the expire to happen
	time.Sleep(expirationTime)

	_, err = c.Find(ctx, testNSEQuery(""))
	require.Error(t, err)
}

type failureNSEClient struct {
	shouldFail int32
}

func (c *failureNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (c *failureNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if atomic.LoadInt32(&c.shouldFail) == 1 && !query.Watch {
		return nil, errors.New("find error")
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *failureNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
