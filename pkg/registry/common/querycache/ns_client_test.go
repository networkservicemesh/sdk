// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
//
// SPDX-Licens-Identifier: Apache-2.0
//
// Licensd under the Apache Licens, Version 2.0 (the "Licens");
// you may not use this file except in compliance with the Licens.
// You may obtain a copy of the Licens at:
//
//     http://www.apache.org/licenss/LICEns-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the Licens is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the Licens for the specific language governing permissions and
// limitations under the Licens.

package querycache_test

import (
	"context"
	"sync/atomic"
	"testing"

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
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	payload1 = "ethernet"
	payload2 = "ip"
)

func testNSQuery(nsName string) *registry.NetworkServiceQuery {
	return &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: nsName,
		},
	}
}

func Test_QueryCacheClient_ShouldCacheNSs(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceRegistryServer()

	failureClient := new(failureNSClient)
	c := next.NewNetworkServiceRegistryClient(
		querycache.NewNetworkServiceClient(ctx, querycache.WithNSExpireTimeout(expireTimeout)),
		failureClient,
		adapters.NetworkServiceServerToClient(mem),
	)

	reg, err := mem.Register(ctx, &registry.NetworkService{
		Name:    name,
		Payload: payload1,
	})
	require.NoError(t, err)

	// Goroutines should be cleaned up on ns unregister
	t.Cleanup(func() { goleak.VerifyNone(t) })

	// 1. Find from memory
	atomic.StoreInt32(&failureClient.shouldFail, 0)

	stream, err := c.Find(ctx, testNSQuery(""))
	require.NoError(t, err)
	nsResp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, name, nsResp.NetworkService.Name)

	// 2. Find from cache
	atomic.StoreInt32(&failureClient.shouldFail, 1)

	stream, err = c.Find(ctx, testNSQuery(name))
	require.NoError(t, err)
	nsResp, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, name, nsResp.NetworkService.Name)

	// 3. Update NS in memory
	reg.Payload = payload2
	reg, err = mem.Register(ctx, reg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		if stream, err = c.Find(ctx, testNSQuery(name)); err != nil {
			return false
		}
		if nsResp, err = stream.Recv(); err != nil {
			return false
		}
		return name == nsResp.NetworkService.Name && payload2 == nsResp.NetworkService.Payload
	}, testWait, testTick)

	// 4. Delete ns from memory
	_, err = mem.Unregister(ctx, reg)
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		_, err = c.Find(ctx, testNSQuery(name))
		return err != nil
	}, testWait, testTick)
}

func Test_QueryCacheClient_ShouldCleanUpNSOnTimeout(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	mem := memory.NewNetworkServiceRegistryServer()

	failureClient := new(failureNSClient)
	c := next.NewNetworkServiceRegistryClient(
		querycache.NewNetworkServiceClient(ctx, querycache.WithNSExpireTimeout(expireTimeout)),
		failureClient,
		adapters.NetworkServiceServerToClient(mem),
	)

	_, err := mem.Register(ctx, &registry.NetworkService{
		Name: name,
	})
	require.NoError(t, err)

	// Goroutines should be cleaned up on cache entry expiration
	t.Cleanup(func() { goleak.VerifyNone(t) })

	// 1. Find from memory
	atomic.StoreInt32(&failureClient.shouldFail, 0)

	stream, err := c.Find(ctx, testNSQuery(""))
	require.NoError(t, err)

	_, err = stream.Recv()
	require.NoError(t, err)

	// 2. Find from cache
	atomic.StoreInt32(&failureClient.shouldFail, 1)

	require.Eventually(t, func() bool {
		if stream, err = c.Find(ctx, testNSQuery(name)); err == nil {
			_, err = stream.Recv()
		}
		return err == nil
	}, testWait, testTick)

	// 3. Keep finding from cache to prevent expiration
	for start := clockMock.Now(); clockMock.Since(start) < 2*expireTimeout; clockMock.Add(expireTimeout / 3) {
		stream, err = c.Find(ctx, testNSQuery(name))
		require.NoError(t, err)

		_, err = stream.Recv()
		require.NoError(t, err)
	}

	// 4. Wait for the expire to happen
	clockMock.Add(expireTimeout)

	_, err = c.Find(ctx, testNSQuery(name))
	require.Errorf(t, err, "find error")
}

type failureNSClient struct {
	shouldFail int32
}

func (c *failureNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
}

func (c *failureNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if atomic.LoadInt32(&c.shouldFail) == 1 && !query.Watch {
		return nil, errors.New("find error")
	}
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *failureNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}
