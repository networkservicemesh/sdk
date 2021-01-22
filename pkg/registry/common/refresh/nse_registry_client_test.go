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
	"sync"
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
)

const testExpiryDuration = time.Millisecond * 100

type testNSEClient struct {
	sync.Mutex
	requestCount int
}

func (t *testNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	t.Lock()
	defer t.Unlock()
	t.requestCount++
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (t *testNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	panic("implement me")
}

func (t *testNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return nil, nil
}

type checkExpirationTimeClient struct{ *testing.T }

func (c *checkExpirationTimeClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	require.NotNil(c, in.ExpirationTime)
	return in, nil
}

func (c *checkExpirationTimeClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	panic("implement me")
}

func (c *checkExpirationTimeClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}

type nseExpirationServer struct {
	expirationTime time.Duration
}

func (s *nseExpirationServer) Register(_ context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	resp := in.Clone()
	resp.ExpirationTime = timestamppb.New(time.Now().Add(s.expirationTime))
	return resp, nil
}

func (s *nseExpirationServer) Find(_ *registry.NetworkServiceEndpointQuery, _ registry.NetworkServiceEndpointRegistry_FindServer) error {
	return nil
}

func (s *nseExpirationServer) Unregister(_ context.Context, _ *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return nil, nil
}

func TestNewNetworkServiceEndpointRegistryClient(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	testClient := &testNSEClient{}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(
			refresh.WithRetryPeriod(testExpiryDuration),
			refresh.WithDefaultExpiryDuration(testExpiryDuration),
			refresh.WithChainContext(ctx),
		),
		testClient)
	_, err := refreshClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})

	require.NoError(t, err)
	require.Eventually(t, func() bool {
		testClient.Lock()
		defer testClient.Unlock()
		return testClient.requestCount > 0
	}, time.Second, testExpiryDuration/4)

	_, err = refreshClient.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
}

func TestRefreshNSEClient_ShouldSetExpirationTime_BeforeCallNext(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := next.NewNetworkServiceEndpointRegistryClient(
		&testNSEClient{},
		refresh.NewNetworkServiceEndpointRegistryClient(refresh.WithDefaultExpiryDuration(time.Hour), refresh.WithChainContext(ctx)),
		&checkExpirationTimeClient{T: t},
	)

	_, err := c.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.Nil(t, err)

	_, err = c.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.Nil(t, err)
}

func Test_RefreshNSEClient_ShouldUseExpirationFromServer(t *testing.T) {
	c := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(refresh.WithDefaultExpiryDuration(time.Hour)),
		adapters.NetworkServiceEndpointServerToClient(&nseExpirationServer{expirationTime: time.Hour}),
	)

	resp, err := c.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})

	require.NoError(t, err)
	require.Greater(t, time.Until(resp.ExpirationTime.AsTime()).Minutes(), float64(50))
}

func Test_RefreshNSEClient_CalledRegisterTwice(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testClient := &testNSEClient{}

	refreshClient := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(
			refresh.WithRetryPeriod(testExpiryDuration),
			refresh.WithDefaultExpiryDuration(testExpiryDuration),
			refresh.WithChainContext(ctx)),
		testClient)

	_, err := refreshClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})

	require.Nil(t, err)
	_, err = refreshClient.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})

	require.Nil(t, err)

	require.Eventually(t, func() bool {
		testClient.Lock()
		defer testClient.Unlock()
		return testClient.requestCount > 0
	}, time.Second, testExpiryDuration/4)

	_, err = refreshClient.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})

	require.Nil(t, err)
}
