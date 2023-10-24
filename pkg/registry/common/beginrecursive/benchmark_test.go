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

package beginrecursive_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginrecursive"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type dataRaceServer struct {
	count int
}

func (s *dataRaceServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.count++
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *dataRaceServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *dataRaceServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.count--
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func BenchmarkBeginRecursive_RegisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBeginRecursive_UnregisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBeginRecursive_RegisterUnregisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(2 * b.N)
	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			go func() {
				server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < b.N; i++ {
			go func() {
				server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}

func BenchmarkBeginRecursive_RegisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		local := i
		go func() {
			server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBeginRecursive_UnregisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		local := i
		go func() {
			server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBeginRecursive_RegisterUnregisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		beginrecursive.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(2 * b.N)
	b.ResetTimer()
	go func() {
		for i := 0; i < b.N; i++ {
			local := i
			go func() {
				server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < b.N; i++ {
			local := i
			go func() {
				server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}
