// Copyright (c) 2023 Cisco and/or its affiliates.
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

package beginrecursive1_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"

	"go.uber.org/goleak"
)

const (
	eventCount = 100
)

func TestServer_ConcurrentUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	datarace := &dataRaceServer{count: 0}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		datarace,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(eventCount)
	for i := 0; i < eventCount; i++ {
		go func() {
			_, err := server.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "1"})
			require.NoError(t, err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, -eventCount, datarace.count)
}

func TestServer_ConcurrentRegister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	datarace := &dataRaceServer{count: 0}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		datarace,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(eventCount)

	for i := 0; i < eventCount; i++ {
		go func() {
			_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{Name: "1"})
			require.NoError(t, err)
			wg.Done()
		}()
	}

	wg.Wait()
	require.Equal(t, eventCount, datarace.count)
}

func TestServer_ConcurrentRegisterUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	datarace := &dataRaceServer{count: 0}
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		datarace,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2 * eventCount)

	go func() {
		for i := 0; i < eventCount; i++ {
			go func() {
				_, err := server.Register(ctx, &registry.NetworkServiceEndpoint{Name: "1"})
				require.NoError(t, err)
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < eventCount; i++ {
			go func() {
				_, err := server.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "1"})
				require.NoError(t, err)
				wg.Done()
			}()
		}
	}()

	wg.Wait()
	require.Equal(t, 0, datarace.count)
}
