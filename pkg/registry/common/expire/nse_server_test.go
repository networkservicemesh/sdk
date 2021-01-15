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

	"go.uber.org/goleak"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

type remoteNSEServer struct{}

func (n *remoteNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return nse.Clone(), nil
}

func (n *remoteNSEServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *remoteNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func Test_ExpireServer_ShouldCorrectlySetExpirationTime_InRemoteCase(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(expire.NewNetworkServiceEndpointRegistryServer(time.Hour), new(remoteNSEServer))

	resp, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})

	require.NoError(t, err)

	require.Greater(t, time.Until(resp.ExpirationTime.AsTime()).Minutes(), float64(50))
}

func TestNewNetworkServiceEndpointRegistryServer(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	s := next.NewNetworkServiceEndpointRegistryServer(
		expire.NewNetworkServiceEndpointRegistryServer(testPeriod*2),
		newCloneEndpointRegistryServer(), // <-- GRPC invocation
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
		require.Nil(t, err)
		list = registry.ReadNetworkServiceEndpointList(stream)
		return len(list) == 0
	}, time.Second, time.Millisecond*100)
}

func Test_ExpireEndpointRegistryServer_ShouldCorrectlyRescheduleTimer(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())

	c := next.NewNetworkServiceEndpointRegistryClient(
		refresh.NewNetworkServiceEndpointRegistryClient(refresh.WithChainContext(ctx)),
		adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
			newCloneEndpointRegistryServer(), // <-- GRPC invocation
			expire.NewNetworkServiceEndpointRegistryServer(testPeriod*2),
			newCloneEndpointRegistryServer(), // <-- GRPC invocation
			memory.NewNetworkServiceEndpointRegistryServer(),
		)))

	_, err := c.Register(context.Background(), &registry.NetworkServiceEndpoint{})
	require.NoError(t, err)

	<-time.After(time.Second)

	stream, err := c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
	})
	require.NoError(t, err)
	list := registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, list, 1)

	cancel()

	require.Eventually(t, func() bool {
		stream, err := c.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
		})
		require.NoError(t, err)
		list := registry.ReadNetworkServiceEndpointList(stream)
		return len(list) == 0
	}, time.Second, time.Millisecond*100)
}

type cloneEndpointRegistryServer struct {
	registry.NetworkServiceEndpointRegistryServer
}

func newCloneEndpointRegistryServer() *cloneEndpointRegistryServer {
	return &cloneEndpointRegistryServer{
		NetworkServiceEndpointRegistryServer: null.NewNetworkServiceEndpointRegistryServer(),
	}
}

func (c *cloneEndpointRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse.Clone())
}
