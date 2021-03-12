// Copyright (c) 2021 Doc.ai and/or its affiliates.
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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/localbypass"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
)

const (
	nsmgrURL = "tcp://0.0.0.0"
	nseURL   = "tcp://1.1.1.1"
)

func TestLocalBypassNSEServer(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 1. Register
	nse, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.Url)

	stream, err := adapters.NetworkServiceEndpointServerToClient(mem).Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	findNSE, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nsmgrURL, findNSE.Url)

	// 2. Find
	stream, err = adapters.NetworkServiceEndpointServerToClient(server).Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	findNSE, err = stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nseURL, findNSE.Url)

	// 3. Unregister
	_, err = server.Unregister(context.Background(), nse)
	require.NoError(t, err)

	stream, err = adapters.NetworkServiceEndpointServerToClient(mem).Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err)
}

func TestLocalBypassNSEServer_SlowRegistry(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		checknse.NewServer(t, func(*testing.T, *registry.NetworkServiceEndpoint) {
			time.Sleep(10 * time.Millisecond)
		}),
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

			nse, err := stream.Recv()
			if err != nil {
				return
			}

			require.Equal(t, nseURL, nse.Url)
		}
	}()

	// 2. Register
	nse, err := server.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  nseURL,
	})
	require.NoError(t, err)
	require.Equal(t, nseURL, nse.Url)

	// 3. Unregister
	_, err = server.Unregister(context.Background(), nse)
	require.NoError(t, err)
}

func TestLocalBypassNSEServer_NotExistingEndpoint(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	server := next.NewNetworkServiceEndpointRegistryServer(
		localbypass.NewNetworkServiceEndpointRegistryServer(nsmgrURL),
		mem,
	)

	// 1. Register directly to the memory
	_, err := mem.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "nse-1",
		Url:  nsmgrURL,
	})
	require.NoError(t, err)

	// 2. Find
	stream, err := adapters.NetworkServiceEndpointServerToClient(server).Find(context.Background(), &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	nse, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, nsmgrURL, nse.Url)
}
