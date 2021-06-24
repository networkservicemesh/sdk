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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	expireTimeout = 3 * time.Minute
	testWait      = 100 * time.Millisecond
	testTick      = testWait / 100
)

func testNSE(clockTime clock.Clock) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name:           "nse-1",
		ExpirationTime: timestamppb.New(clockTime.Now().Add(expireTimeout)),
	}
}

func TestNewNetworkServiceEndpointRegistryClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		serialize.NewNetworkServiceEndpointRegistryClient(),
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
		serialize.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		countClient,
	)

	reg, err := client.Register(ctx, testNSE(clockMock))
	require.NoError(t, err)

	reg, err = client.Register(ctx, reg)
	require.NoError(t, err)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 2
	}, testWait, testTick)

	_, err = client.Unregister(ctx, reg)
	require.NoError(t, err)
}

func Test_RefreshNSEClient_SetsCorrectExpireTime(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		serialize.NewNetworkServiceEndpointRegistryClient(),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx),
		checknse.NewClient(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			require.Equal(t, expireTimeout, clockMock.Until(nse.ExpirationTime.AsTime().Local()))
		}),
		countClient,
	)

	reg, err := client.Register(ctx, testNSE(clockMock))
	require.NoError(t, err)

	for i := 1; i <= 3; i++ {
		count := int32(i)

		clockMock.Add(expireTimeout / 3 * 2)
		require.Eventually(t, func() bool {
			return atomic.LoadInt32(&countClient.requestCount) > count
		}, testWait, testTick)

		// Wait for the Refresh to fully happen
		time.Sleep(testWait)
	}

	reg.ExpirationTime = timestamppb.New(clockMock.Now().Add(expireTimeout))

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
