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

package querycache_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/querycache"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type FindCountServer struct{ findCount *int32 }

func (a *FindCountServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (a *FindCountServer) Find(q *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	if !q.Watch {
		atomic.AddInt32(a.findCount, 1)
	}
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(q, s)
}

func (a *FindCountServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func Test_QueryCacheServer_ShouldCacheNSEs(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	findsCount := new(int32)
	const registersCount = 5

	mem := next.NewNetworkServiceEndpointRegistryServer(&FindCountServer{findCount: findsCount}, memory.NewNetworkServiceEndpointRegistryServer())

	for i := 0; i < registersCount; i++ {
		_, err := mem.Register(ctx, &registry.NetworkServiceEndpoint{
			Name: fmt.Sprintf("nse-%v", i),
		})
		require.NoError(t, err)
	}

	client := next.NewNetworkServiceEndpointRegistryClient(querycache.NewClient(ctx), adapters.NetworkServiceEndpointServerToClient(mem))
	_, err := client.Find(ctx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{}})
	require.NoError(t, err)

	checkCache := func() {
		for i := 0; i < registersCount; i++ {
			name := fmt.Sprintf("nse-%v", i)
			stream, err := client.Find(ctx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
			})
			require.NoError(t, err)
			list := registry.ReadNetworkServiceEndpointList(stream)
			require.Len(t, list, 1)
			require.Equal(t, name, list[0].Name)
		}
		n := atomic.LoadInt32(findsCount)
		require.Equal(t, int32(1), n)
	}

	for i := 0; i < 1e3; i++ {
		checkCache()
	}
	for i := 0; i < registersCount; i++ {
		name := fmt.Sprintf("nse-%v", i)
		_, err := mem.Unregister(ctx, &registry.NetworkServiceEndpoint{
			Name: name,
		})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			stream, err := client.Find(ctx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
			})
			require.NoError(t, err)
			list := registry.ReadNetworkServiceEndpointList(stream)
			return len(list) == 0
		}, time.Second, time.Second/10)
	}
}

func Test_QueryCacheServer_ShouldCleanupGoroutinesOnNSEUnregister(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	reg, err := func() (*registry.NetworkServiceEndpoint, error) {
		registerCtx, registerCancel := context.WithCancel(ctx)
		defer registerCancel()

		return mem.Register(registerCtx, &registry.NetworkServiceEndpoint{
			Name: "nse-1",
		})
	}()
	require.NoError(t, err)

	client := next.NewNetworkServiceEndpointRegistryClient(
		querycache.NewClient(ctx),
		adapters.NetworkServiceEndpointServerToClient(mem),
	)

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	// 1. Find
	findCtx, findCancel := context.WithCancel(ctx)

	_, err = client.Find(findCtx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: reg.Name,
		},
	})
	require.NoError(t, err)

	findCancel()

	// 2. Wait a bit for the (cache -> registry) stream to start
	<-time.After(1 * time.Millisecond)

	// 3. Unregister
	unregisterCtx, unregisterCancel := context.WithCancel(ctx)

	_, err = mem.Unregister(unregisterCtx, reg)
	require.NoError(t, err)

	unregisterCancel()
}
