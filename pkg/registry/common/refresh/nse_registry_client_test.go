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

package refresh_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
	"github.com/networkservicemesh/sdk/pkg/tools/interdomain"
)

const (
	expireTimeout = 3 * time.Minute
	testWait      = 100 * time.Millisecond
	testTick      = testWait / 100
)

type injectNSERegisterClient struct {
	registry.NetworkServiceEndpointRegistryClient
	register func(context.Context, *registry.NetworkServiceEndpoint, ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error)
}

func (c *injectNSERegisterClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return c.register(ctx, nse, opts...)
}

func testNSE(clockTime clock.Clock) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:           "nse-1",
		ExpirationTime: timestamppb.New(clockTime.Now().Add(expireTimeout)),
	}
}

func Test_NetworkServiceEndpointRefreshClient_ShouldWorkCorrectlyWithFloatingScenario(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	var registerCount int32

	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		&injectNSERegisterClient{
			NetworkServiceEndpointRegistryClient: null.NewNetworkServiceEndpointRegistryClient(),
			register: func(c context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
				resp := nse.Clone()
				resp.Name = interdomain.Target(nse.GetName())
				require.Equal(t, "nse-1@my.domain.com", nse.GetName())
				atomic.AddInt32(&registerCount, 1)
				return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, resp, opts...)
			},
		},
	)

	nse := testNSE(clockMock)
	nse.Name = "nse-1@my.domain.com"
	reg, err := client.Register(ctx, nse)
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&registerCount) > 1
	}, testWait, testTick)

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

func TestNewNetworkServiceEndpointRegistryClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		countClient,
	)

	reg, err := client.Register(ctx, testNSE(clockMock))
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 1
	}, testWait, testTick)

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

func Test_RefreshNSEClient_CalledRegisterTwice(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		countClient,
	)

	reg, err := client.Register(context.Background(), testNSE(clockMock))
	require.NoError(t, err)

	reg, err = client.Register(context.Background(), reg)
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 2
	}, testWait, testTick)

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

func Test_RefreshNSEClient_StopsRefreshOnUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	goleak.VerifyNone(t)

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	ignoreClockMockGoroutine := goleak.IgnoreCurrent()

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		countClient,
	)

	const registerCount = 100

	var regs []*registry.NetworkServiceEndpoint

	for i := 0; i < registerCount; i++ {
		regs = append(regs, testNSE(clockMock))
		regs[i].Name = fmt.Sprint(i)
		resp, err := client.Register(ctx, regs[i])
		require.NoError(t, err)
		regs[i] = resp
		regs[i].ExpirationTime = timestamppb.New(clockMock.Now().Add(expireTimeout))
	}

	clockMock.Add(expireTimeout / 3 * 2)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) >= 2*int32(len(regs))
	}, testWait, testTick)

	for i := 0; i < registerCount; i++ {
		_, err := client.Unregister(ctx, regs[i])
		require.NoError(t, err)
	}

	goleak.VerifyNone(t, ignoreClockMockGoroutine)

	for i := 0; i < 5; i++ {
		clockMock.Add(expireTimeout / 3 * 2)

		require.Never(t, func() bool {
			return atomic.LoadInt32(&countClient.requestCount) > registerCount*3
		}, testWait, testTick)

		// Wait for the Refresh to fully happen
		time.Sleep(testWait)
	}
}

func Test_RefreshNSEClient_SetsCorrectExpireTime(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		countClient,
		checknse.NewClient(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			nse.ExpirationTime = testNSE(clockMock).GetExpirationTime()
		}),
	)

	reg, err := client.Register(ctx, testNSE(clockMock))
	require.NoError(t, err)

	for i := 1; i <= 3; i++ {
		count := int32(i)

		clockMock.Add(expireTimeout / 3 * 2)
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&countClient.requestCount) > count
		}, 4*testWait, testTick)

		// Wait for the Refresh to fully happen
		time.Sleep(testWait)
	}

	reg.ExpirationTime = timestamppb.New(clockMock.Now().Add(expireTimeout))

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

func Test_RefreshNSEClient_CorrectInitialRegTime(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)
	regTime := clockMock.Now()

	var registerCount int32

	client := next.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		&injectNSERegisterClient{
			NetworkServiceEndpointRegistryClient: null.NewNetworkServiceEndpointRegistryClient(),
			register: func(c context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
				require.NotNil(t, nse.GetInitialRegistrationTime())
				require.True(t, nse.GetInitialRegistrationTime().AsTime().Equal(regTime))
				atomic.AddInt32(&registerCount, 1)
				return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
			},
		},
	)

	nse := testNSE(clockMock)
	nse.InitialRegistrationTime = timestamppb.New(regTime)
	reg, err := client.Register(ctx, nse)
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&registerCount) > 1
	}, testWait, testTick)

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

type requestCountClient struct {
	requestCount int32

	registry.NetworkServiceEndpointRegistryClient
}

func (t *requestCountClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	atomic.AddInt32(&t.requestCount, 1)

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (t *requestCountClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
