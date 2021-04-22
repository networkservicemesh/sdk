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
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/expire"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	expireTimeout = time.Minute
	nsName        = "ns"
	testWait      = 100 * time.Millisecond
	testTick      = testWait / 100
)

func TestExpireNSServer_NSE_Expired(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	nseMem := memory.NewNetworkServiceEndpointRegistryServer()
	nsMem := memory.NewNetworkServiceRegistryServer()

	updateServer := new(updateNSEServer)

	s := next.NewNetworkServiceRegistryServer(
		expire.NewNetworkServiceServer(
			ctx,
			adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
				updateServer,
				nseMem,
			))),
		nsMem,
	)

	_, err := s.Register(ctx, &registry.NetworkService{
		Name: nsName,
	})
	require.NoError(t, err)

	names := make([]string, 10)
	for i := 0; i < len(names); i++ {
		names[i] = fmt.Sprint("nse-", i)
		_, err = nseMem.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                names[i],
			NetworkServiceNames: []string{nsName},
			ExpirationTime:      timestamppb.New(clockMock.Now().Add(expireTimeout)),
		})
		require.NoError(t, err)
	}

	// Wait for the update from nseMem
	require.Eventually(t, func() bool {
		for _, name := range names {
			if _, ok := updateServer.updates.Load(name); !ok {
				return false
			}
		}
		return true
	}, testWait, testTick)

	c := adapters.NetworkServiceServerToClient(nsMem)

	stream, err := c.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
	})
	require.NoError(t, err)

	ns, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nsName, ns.Name)

	clockMock.Add(expireTimeout)
	require.Eventually(t, func() bool {
		stream, err = c.Find(ctx, &registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
		})
		require.NoError(t, err)

		_, err = stream.Recv()
		return err == io.EOF
	}, testWait, testTick)
}

func TestExpireNSServer_NSE_Unregistered(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	nseMem := memory.NewNetworkServiceEndpointRegistryServer()
	nsMem := memory.NewNetworkServiceRegistryServer()

	updateServer := new(updateNSEServer)

	s := next.NewNetworkServiceRegistryServer(
		expire.NewNetworkServiceServer(
			ctx,
			adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
				updateServer,
				nseMem,
			))),
		nsMem,
	)

	_, err := s.Register(ctx, &registry.NetworkService{
		Name: nsName,
	})
	require.NoError(t, err)

	names := make([]string, 10)
	for i := 0; i < len(names); i++ {
		names[i] = fmt.Sprint("nse-", i)
		_, err = nseMem.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                names[i],
			NetworkServiceNames: []string{nsName},
			ExpirationTime:      timestamppb.New(clockMock.Now().Add(expireTimeout)),
		})
		require.NoError(t, err)
	}

	// Wait for the update from nseMem
	require.Eventually(t, func() bool {
		for _, name := range names {
			if _, ok := updateServer.updates.Load(name); !ok {
				return false
			}
		}
		return true
	}, testWait, testTick)

	c := adapters.NetworkServiceServerToClient(nsMem)

	stream, err := c.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
	})
	require.NoError(t, err)

	ns, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nsName, ns.Name)

	for i := 0; i < 10; i++ {
		_, err = nseMem.Unregister(ctx, &registry.NetworkServiceEndpoint{
			Name: fmt.Sprint("nse-", i),
		})
		require.NoError(t, err)
	}

	require.Eventually(t, func() bool {
		stream, err = c.Find(ctx, &registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
		})
		require.NoError(t, err)

		_, err = stream.Recv()
		return err == io.EOF
	}, testWait, testTick)
}

func TestExpireNSServer_NSE_Update(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	const nseName = "nse"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.NewMock()
	ctx = clock.WithClock(ctx, clockMock)

	nseMem := memory.NewNetworkServiceEndpointRegistryServer()
	nsMem := memory.NewNetworkServiceRegistryServer()

	updateServer := new(updateNSEServer)

	s := next.NewNetworkServiceRegistryServer(
		expire.NewNetworkServiceServer(
			ctx,
			adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
				updateServer,
				nseMem,
			))),
		nsMem,
	)

	_, err := s.Register(ctx, &registry.NetworkService{
		Name: nsName,
	})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		updateServer.updates = sync.Map{}

		_, err = nseMem.Register(ctx, &registry.NetworkServiceEndpoint{
			Name:                nseName,
			NetworkServiceNames: []string{nsName},
			ExpirationTime:      timestamppb.New(clockMock.Now().Add(expireTimeout)),
		})
		require.NoError(t, err)

		// Wait for the update from nseMem
		require.Eventually(t, func() bool {
			_, ok := updateServer.updates.Load(nseName)
			return ok
		}, testWait, testTick)

		c := adapters.NetworkServiceServerToClient(nsMem)

		stream, err := c.Find(ctx, &registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
		})
		require.NoError(t, err)

		ns, err := stream.Recv()
		require.NoError(t, err)
		require.Equal(t, nsName, ns.Name)

		clockMock.Add(expireTimeout / 2)
	}
}

type updateNSEServer struct {
	updates sync.Map
}

func (s *updateNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (s *updateNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &updateNSEFindServer{
		updateNSEServer: s,
		NetworkServiceEndpointRegistry_FindServer: server,
	})
}

func (s *updateNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*emptypb.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

type updateNSEFindServer struct {
	*updateNSEServer
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *updateNSEFindServer) Send(nse *registry.NetworkServiceEndpoint) error {
	s.updates.Store(nse.Name, struct{}{})
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nse)
}
