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
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
)

const testExpiryDuration = time.Millisecond * 100

func testNSE() *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	}
}

func TestNewNetworkServiceEndpointRegistryClient(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(ctx, refresh.WithDefaultExpiryDuration(testExpiryDuration)),
		countClient,
	)

	_, err := client.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	})
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 1
	}, time.Second, testExpiryDuration/4)

	_, err = client.Unregister(context.Background(), testNSE())
	require.NoError(t, err)
}

func TestRefreshNSEClient_ShouldSetExpirationTime_BeforeCallNext(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := next.NewNetworkServiceEndpointRegistryClient(
		new(requestCountClient),
		refresh.NewNetworkServiceEndpointRegistryClient(ctx, refresh.WithDefaultExpiryDuration(time.Hour)),
		checknse.NewClient(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			require.NotNil(t, nse.ExpirationTime)
		}),
	)

	reg, err := client.Register(context.Background(), testNSE())
	require.NoError(t, err)

	_, err = client.Unregister(context.Background(), reg)
	require.Nil(t, err)
}

func Test_RefreshNSEClient_CalledRegisterTwice(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	countClient := new(requestCountClient)
	client := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(ctx, refresh.WithDefaultExpiryDuration(testExpiryDuration)),
		countClient,
	)

	_, err := client.Register(context.Background(), testNSE())
	require.NoError(t, err)

	reg, err := client.Register(context.Background(), testNSE())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 2
	}, time.Second, testExpiryDuration/4)

	_, err = client.Unregister(context.Background(), reg)
	require.NoError(t, err)
}

func Test_RefreshNSEClient_ShouldOverrideNameAndDuration(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	endpoint := &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  "url",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	countClient := new(requestCountClient)
	registryServer := &nseRegistryServer{
		name:           uuid.New().String(),
		expiryDuration: testExpiryDuration,
	}
	client := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(ctx, refresh.WithDefaultExpiryDuration(time.Hour)),
		checknse.NewClient(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			if countClient.requestCount > 0 {
				require.Equal(t, registryServer.name, nse.Name)
				require.Equal(t, endpoint.Url, nse.Url)
			}
		}),
		countClient,
		adapters.NetworkServiceEndpointServerToClient(registryServer),
	)

	reg, err := client.Register(context.Background(), endpoint.Clone())
	require.NoError(t, err)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&countClient.requestCount) > 3
	}, time.Second, testExpiryDuration/4)

	reg.Url = endpoint.Url

	_, err = client.Unregister(context.Background(), reg)
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

type nseRegistryServer struct {
	expiryDuration time.Duration
	name           string

	registry.NetworkServiceEndpointRegistryServer
}

func (s *nseRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	nse = nse.Clone()
	nse.Name = s.name
	nse.Url = uuid.New().String()
	nse.ExpirationTime = timestamppb.New(time.Now().Add(s.expiryDuration))

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *nseRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
