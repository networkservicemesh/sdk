// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package proxy_test

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
)

func TestNewProxyNetworkServiceEndpointRegistryServer_Register(t *testing.T) {
	m := memory.NewNetworkServiceEndpointRegistryServer()
	u, closeServer := startNSEServer(t, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chain := testingNSEServerChain(ctx, u)

	_, err := chain.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
	_, err = chain.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-2@domain"})
	require.NoError(t, err)
	_, err = chain.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-3"})
	require.NoError(t, err)

	client := adapters.NetworkServiceEndpointServerToClient(m)

	stream, err := client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse"}})
	require.NoError(t, err)
	list := registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, list, 1)
	require.Equal(t, "nse-2@domain", list[0].Name)

	closeServer()

	require.Eventually(t, func() bool {
		runtime.GC()
		return goleak.Find() != nil
	}, time.Second, time.Microsecond*100)
}

func TestNewProxyNetworkServiceEndpointRegistryServer_Unregister(t *testing.T) {
	m := memory.NewNetworkServiceEndpointRegistryServer()
	_, err := m.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"})
	require.Nil(t, err)
	u, closeServer := startNSEServer(t, m)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chain := testingNSEServerChain(ctx, u)

	checkLen := func(expected int) {
		client := adapters.NetworkServiceEndpointServerToClient(m)
		stream, findErr := client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse"}})
		require.NoError(t, findErr)
		list := registry.ReadNetworkServiceEndpointList(stream)
		require.Len(t, list, expected)
	}

	_, err = chain.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)
	checkLen(1)
	_, err = chain.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse"})
	require.NoError(t, err)
	checkLen(1)
	_, err = chain.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1@domain2"})
	require.NoError(t, err)
	checkLen(1)
	_, err = chain.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"})
	require.NoError(t, err)
	checkLen(0)

	closeServer()

	require.Eventually(t, func() bool {
		runtime.GC()
		return goleak.Find() != nil
	}, time.Second, time.Microsecond*100)
}

func TestNewProxyNetworkServiceEndpointRegistryServer_Find(t *testing.T) {
	m := memory.NewNetworkServiceEndpointRegistryServer()
	_, err := m.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "nse-1@domain1"})
	require.Nil(t, err)
	u, closeServer := startNSEServer(t, m)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chain := testingNSEServerChain(ctx, u)

	checkLen := func(nseName string, expected int) {
		client := adapters.NetworkServiceEndpointServerToClient(chain)
		stream, err := client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: nseName}})
		require.NoError(t, err)
		list := registry.ReadNetworkServiceEndpointList(stream)
		require.Len(t, list, expected)
	}

	checkLen("nse", 0)
	checkLen("nse-1@domain1", 1)

	closeServer()
	require.Eventually(t, func() bool {
		runtime.GC()
		return true
	}, time.Second, time.Microsecond*100)
}
