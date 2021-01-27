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

package expire_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func TestExpireNSEServer_ShouldCorrectlySetExpirationTime_InRemoteCase(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(
		expire.NewNetworkServiceEndpointRegistryServer(time.Hour),
		new(remoteNSEServer),
	)

	resp, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	require.Greater(t, time.Until(resp.ExpirationTime.AsTime()).Minutes(), float64(50))
}

func TestExpireNSEServer_ShouldUseLessExpirationTimeFromInput_AndWork(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(expire.NewNetworkServiceEndpointRegistryServer(time.Hour), memory.NewNetworkServiceEndpointRegistryServer())

	resp, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name:           "nse-1",
		ExpirationTime: timestamppb.New(time.Now().Add(time.Millisecond * 200)),
	})
	require.NoError(t, err)

	require.Less(t, time.Until(resp.ExpirationTime.AsTime()).Seconds(), float64(65))

	c := adapters.NetworkServiceEndpointServerToClient(s)

	require.Eventually(t, func() bool {
		stream, err := c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
		})
		require.NoError(t, err)

		list := registry.ReadNetworkServiceEndpointList(stream)
		return len(list) == 0
	}, time.Second, time.Millisecond*100)
}

func TestExpireNSEServer_ShouldUseLessExpirationTimeFromResponse(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(
		expire.NewNetworkServiceEndpointRegistryServer(time.Hour),
		new(remoteNSEServer), // <-- GRPC invocation
		expire.NewNetworkServiceEndpointRegistryServer(10*time.Minute),
	)

	resp, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	require.Less(t, time.Until(resp.ExpirationTime.AsTime()).Minutes(), float64(11))
}

func TestExpireNSEServer_ShouldRemoveNSEAfterExpirationTime(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(
		expire.NewNetworkServiceEndpointRegistryServer(testPeriod*2),
		new(remoteNSEServer), // <-- GRPC invocation
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	_, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{})
	require.NoError(t, err)

	c := adapters.NetworkServiceEndpointServerToClient(s)
	stream, err := c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)
	require.NotEmpty(t, list)

	require.Eventually(t, func() bool {
		stream, err = c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
		})
		require.NoError(t, err)

		list = registry.ReadNetworkServiceEndpointList(stream)
		return len(list) == 0
	}, time.Second, time.Millisecond*100)
}

type remoteNSEServer struct {
	registry.NetworkServiceEndpointRegistryServer
}

func (s *remoteNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse.Clone())
}

func (s *remoteNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *remoteNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
