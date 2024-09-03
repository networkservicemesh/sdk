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

package begin_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

var count = 1000

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

func BenchmarkBegin_RegisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(count)
	b.ResetTimer()
	for i := 0; i < count; i++ {
		go func() {
			_, _ = server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBegin_UnregisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(count)
	b.ResetTimer()
	for i := 0; i < count; i++ {
		go func() {
			_, _ = server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBegin_RegisterUnregisterSameIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		&dataRaceServer{count: 0},
	)

	var wg sync.WaitGroup
	wg.Add(2 * count)
	b.ResetTimer()
	go func() {
		for i := 0; i < count; i++ {
			go func() {
				_, _ = server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < count; i++ {
			go func() {
				_, _ = server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: "1"})
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}

func BenchmarkBegin_RegisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
	)

	var wg sync.WaitGroup
	wg.Add(count)
	b.ResetTimer()
	for i := 0; i < count; i++ {
		local := i
		go func() {
			_, _ = server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBegin_UnregisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
	)

	var wg sync.WaitGroup
	wg.Add(count)
	b.ResetTimer()
	for i := 0; i < count; i++ {
		local := i
		go func() {
			_, _ = server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkBegin_RegisterUnregisterDifferentIDs(b *testing.B) {
	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
	)

	var wg sync.WaitGroup
	wg.Add(2 * count)
	b.ResetTimer()
	go func() {
		for i := 0; i < count; i++ {
			local := i
			go func() {
				_, _ = server.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
				wg.Done()
			}()
		}
	}()

	go func() {
		for i := 0; i < count; i++ {
			local := i
			go func() {
				_, _ = server.Unregister(context.Background(), &registry.NetworkServiceEndpoint{Name: fmt.Sprint(local)})
				wg.Done()
			}()
		}
	}()

	wg.Wait()
}
