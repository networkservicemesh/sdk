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

package swapip_test

import (
	"context"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/swapip"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checknse"
)

func TestSwapIPNSERegistryServer_Register(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ipMapCh := make(chan map[string]string)
	s := chain.NewNetworkServiceEndpointRegistryServer(
		swapip.NewNetworkServiceEndpointRegistryServer(ipMapCh),
		checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			require.Equal(t, "tcp://8.8.8.8:5001", nse.GetUrl())
		}),
	)

	event := map[string]string{
		"127.0.0.1": "8.8.8.8",
		"8.8.8.8":   "127.0.0.1",
	}

	ipMapCh <- event
	ipMapCh <- event

	defer close(ipMapCh)

	resp, err := s.Register(ctx, &registry.NetworkServiceEndpoint{
		Url: "tcp://127.0.0.1:5001",
	})
	require.NoError(t, err)
	require.Equal(t, "tcp://127.0.0.1:5001", resp.GetUrl())
}

func TestSwapIPNSERegistryServer_Unregister(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ipMapCh := make(chan map[string]string)

	s := chain.NewNetworkServiceEndpointRegistryServer(
		swapip.NewNetworkServiceEndpointRegistryServer(ipMapCh),
		checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			require.Equal(t, "tcp://8.8.8.8:5001", nse.GetUrl())
		}),
	)
	defer close(ipMapCh)

	event := map[string]string{
		"127.0.0.1": "8.8.8.8",
		"8.8.8.8":   "127.0.0.1",
	}

	ipMapCh <- event
	ipMapCh <- event

	_, err := s.Unregister(ctx, &registry.NetworkServiceEndpoint{
		Url: "tcp://127.0.0.1:5001",
	})
	require.NoError(t, err)
}

func TestSwapIPNSERegistryServer_Find(t *testing.T) {
	defer goleak.VerifyNone(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ipMapCh := make(chan map[string]string)

	m := memory.NewNetworkServiceEndpointRegistryServer()

	_, err := m.Register(ctx, &registry.NetworkServiceEndpoint{
		Url: "tcp://8.8.8.8:5001",
	})
	require.NoError(t, err)

	s := chain.NewNetworkServiceEndpointRegistryServer(
		swapip.NewNetworkServiceEndpointRegistryServer(ipMapCh),
		checknse.NewServer(t, func(t *testing.T, nse *registry.NetworkServiceEndpoint) {
			require.Equal(t, "tcp://8.8.8.8:5001", nse.GetUrl())
		}),
		m,
	)
	defer close(ipMapCh)

	event := map[string]string{
		"127.0.0.1": "8.8.8.8",
		"8.8.8.8":   "127.0.0.1",
	}

	ipMapCh <- event
	ipMapCh <- event

	stream, err := adapters.NetworkServiceEndpointServerToClient(s).Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Url: "tcp://127.0.0.1:5001",
		},
	})
	require.NoError(t, err)

	list := registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, list, 1)

	require.Equal(t, "tcp://127.0.0.1:5001", list[0].GetUrl())
}
