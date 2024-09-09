// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

package localbypass_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

const (
	nsmgrURL = "tcp://0.0.0.0"
	nseURL   = "tcp://1.1.1.1"
)

func TestLocalBypassNSEServer(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 1. Register
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Url: nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.GetUrl())

	stream, err := adapters.NetworkServiceEndpointServerToClient(mem).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nsmgrURL, nseResp.GetNetworkServiceEndpoint().GetUrl())

	// 2. Find
	stream, err = adapters.NetworkServiceEndpointServerToClient(server).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl())

	// 3. Unregister
	_, err = server.Unregister(ctx, nse)
	require.NoError(t, err)

	stream, err = adapters.NetworkServiceEndpointServerToClient(mem).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
}

func TestLocalBypassNSEServer_Restart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 1. Register
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.GetUrl())

	// 2. Find
	stream, err := adapters.NetworkServiceEndpointServerToClient(server).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl())

	// 3. Restart
	server = next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 4. Try to find again
	stream, err = adapters.NetworkServiceEndpointServerToClient(server).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err = stream.Recv()
	if err != nil {
		require.Error(t, err, io.EOF) // either return nothing
	} else {
		require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl()) // or return valid
	}

	// 5. Refresh register
	_, err = server.Register(ctx, nse)
	require.NoError(t, err)

	// 6. Find
	stream, err = adapters.NetworkServiceEndpointServerToClient(server).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl())
}

func TestLocalBypassNSEServer_RegisterNSESameName(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 1. Register
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.GetUrl())

	// 2. NSE restarted with the same name but different URL
	nseRestartedURL := "tcp://1.1.1.2"
	nse, err = server.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse",
		Url:  nseRestartedURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseRestartedURL, nse.GetUrl())

	// 3. Find
	stream, err := adapters.NetworkServiceEndpointServerToClient(server).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nseResp, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nseRestartedURL, nseResp.GetNetworkServiceEndpoint().GetUrl())

	// 4. Unregister
	_, err = server.Unregister(ctx, nse)
	require.NoError(t, err)

	stream, err = adapters.NetworkServiceEndpointServerToClient(mem).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
}

func TestLocalBypassNSEServer_SlowRegistryFind(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		&slowRegistry{
			delay:                                100 * time.Millisecond,
			NetworkServiceEndpointRegistryServer: memory.NewNetworkServiceEndpointRegistryServer(),
		},
	)

	// 1. Start trying to find endpoint
	c := adapters.NetworkServiceEndpointServerToClient(server)
	go func() {
		for ctx.Err() == nil {
			stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
			})
			if err != nil {
				return
			}

			nseResp, err := stream.Recv()
			if err != nil {
				return
			}

			require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl())
		}
	}()

	// 2. Register
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Url: nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.GetUrl())

	// 3. Unregister
	_, err = server.Unregister(ctx, nse)
	require.NoError(t, err)
}

type nsmgrProxyRegistryServer struct {
	registry.NetworkServiceEndpointRegistryServer
}

func (n *nsmgrProxyRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (reg *registry.NetworkServiceEndpoint, err error) {
	nse.Url = "tcp://nsmgr-proxy:5005"
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (n *nsmgrProxyRegistryServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func TestLocalByPass_ShouldCorrectlyHandleNSEsFromFloatingRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	const expectedURL = "file://nse.sock"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	nsmgrRegistryClient := adapters.NetworkServiceEndpointServerToClient(next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		memory.NewNetworkServiceEndpointRegistryServer(),
		new(nsmgrProxyRegistryServer),
	))

	_, err := nsmgrRegistryClient.Register(ctx, &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  expectedURL,
	})

	require.NoError(t, err)

	stream, err := nsmgrRegistryClient.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse-1",
		},
	})

	require.NoError(t, err)

	l := registry.ReadNetworkServiceEndpointList(stream)

	require.Len(t, l, 1)
	require.Equal(t, expectedURL, l[0].GetUrl())
}

func TestLocalBypassNSEServer_SlowRegistryFindWatch(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		&slowRegistry{
			delay:                                100 * time.Millisecond,
			NetworkServiceEndpointRegistryServer: memory.NewNetworkServiceEndpointRegistryServer(),
		},
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// 1. Start watching endpoint
	c := adapters.NetworkServiceEndpointServerToClient(server)
	go func() {
		stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
			Watch:                  true,
		})
		require.NoError(t, err)

		// Register update
		nseResp, err := stream.Recv()
		require.NoError(t, err)

		require.Equal(t, nseURL, nseResp.GetNetworkServiceEndpoint().GetUrl())

		// Unregister update
		nseResp, err = stream.Recv()
		require.NoError(t, err)

		require.True(t, nseResp.GetDeleted())
	}()

	// 2. Register
	nse, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
		Url: nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.GetUrl())

	// 3. Unregister
	_, err = server.Unregister(ctx, nse)
	require.NoError(t, err)

	<-ctx.Done()
}

type slowRegistry struct {
	delay time.Duration

	registry.NetworkServiceEndpointRegistryServer
}

func (r *slowRegistry) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	clock.FromContext(ctx).Sleep(r.delay)
	defer clock.FromContext(ctx).Sleep(r.delay)

	return r.NetworkServiceEndpointRegistryServer.Register(ctx, nse)
}

func (r *slowRegistry) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	clock.FromContext(ctx).Sleep(r.delay)
	defer clock.FromContext(ctx).Sleep(r.delay)

	return r.NetworkServiceEndpointRegistryServer.Unregister(ctx, nse)
}
