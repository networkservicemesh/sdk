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

package refresh_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	testExpiryDuration = time.Millisecond * 100
)

func testNetworkServiceEndpointRegistryClient(ctx context.Context, durations ...time.Duration) registry.NetworkServiceEndpointRegistryClient {
	options := []refresh.Option{refresh.WithChainContext(ctx)}
	if len(durations) > 0 {
		options = append(options, refresh.WithDefaultExpiryDuration(durations[0]))
	}
	if len(durations) > 1 {
		options = append(options, refresh.WithRetryPeriod(durations[1]))
	}
	return refresh.NewNetworkServiceEndpointRegistryClient(options...)
}

func TestRefreshNSEClient(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	testClient := newTestNSEClient()
	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		testNetworkServiceEndpointRegistryClient(context.Background(), testExpiryDuration),
		testClient,
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := refreshClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	cancel()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&testClient.requestCount) > 1
	}, time.Second, testExpiryDuration/4)

	_, err = refreshClient.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
}

func TestRefreshNSEClient_ShouldSetExpirationTime_BeforeCallNext(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	c := next.NewNetworkServiceEndpointRegistryClient(
		newTestNSEClient(),
		testNetworkServiceEndpointRegistryClient(context.Background(), time.Hour),
		newCheckExpirationTimeClient(t),
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := c.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	cancel()
	require.NoError(t, err)

	_, err = c.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
}

func TestRefreshNSEClient_CalledRegisterTwice(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	testClient := newTestNSEClient()
	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		testNetworkServiceEndpointRegistryClient(context.Background(), testExpiryDuration),
		testClient,
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := refreshClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	cancel()
	require.NoError(t, err)

	ctx, cancel = context.WithCancel(context.Background())
	_, err = refreshClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	cancel()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&testClient.requestCount) > 2
	}, time.Second, testExpiryDuration/4)

	_, err = refreshClient.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
}

func TestRefreshNSEClient_RefreshFailsFirstTime(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	testClient := newTestNSEClient()
	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		testNetworkServiceEndpointRegistryClient(context.Background(), testExpiryDuration, testExpiryDuration),
		newFailTimesClient(1),
		testClient,
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := refreshClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	cancel()
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&testClient.requestCount) > 1
	}, time.Second, testExpiryDuration/4)

	_, err = refreshClient.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
}

func TestRefreshNSEClient_RetriesShouldStop(t *testing.T) {
	ignoreCurrent := goleak.IgnoreCurrent()

	bckgCtx, bckgCancel := context.WithCancel(context.Background())
	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		testNetworkServiceEndpointRegistryClient(bckgCtx, testExpiryDuration, time.Hour),
		newFailTimesClient(1, -1),
	)

	ctx, cancel := context.WithCancel(context.Background())
	_, err := refreshClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	cancel()
	require.NoError(t, err)

	<-time.After(testExpiryDuration)
	bckgCancel()

	require.Eventually(t, func() bool {
		return goleak.Find(ignoreCurrent) == nil
	}, time.Second, testExpiryDuration/4)
}

type testNSEClient struct {
	requestCount int32

	registry.NetworkServiceEndpointRegistryClient
}

func newTestNSEClient() *testNSEClient {
	return &testNSEClient{
		NetworkServiceEndpointRegistryClient: null.NewNetworkServiceEndpointRegistryClient(),
	}
}

func (t *testNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	atomic.AddInt32(&t.requestCount, 1)

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

type checkExpirationTimeClient struct {
	*testing.T
	registry.NetworkServiceEndpointRegistryClient
}

func newCheckExpirationTimeClient(t *testing.T) *checkExpirationTimeClient {
	return &checkExpirationTimeClient{
		T:                                    t,
		NetworkServiceEndpointRegistryClient: null.NewNetworkServiceEndpointRegistryClient(),
	}
}

func (c *checkExpirationTimeClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	require.NotNil(c, in.ExpirationTime)

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

type failTimesClient struct {
	times []int
	count int

	registry.NetworkServiceEndpointRegistryClient
}

func newFailTimesClient(times ...int) *failTimesClient {
	return &failTimesClient{
		times:                                times,
		NetworkServiceEndpointRegistryClient: null.NewNetworkServiceEndpointRegistryClient(),
	}
}

func (c *failTimesClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	defer func() { c.count++ }()
	for _, t := range c.times {
		if t > c.count {
			break
		}
		if t == c.count || t == -1 {
			return nil, errors.Errorf("time = %d", t)
		}
	}

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}
