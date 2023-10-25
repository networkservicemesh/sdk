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

package beginmutex_test

import (
	"context"

	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginmutex"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/emptypb"
)

type randomServer struct {
	once sync.Once
}

func (s *randomServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.once.Do(func() { time.Sleep(time.Second) })
	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
	return resp, err
}

func (s *randomServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *randomServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	_, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
	return &emptypb.Empty{}, err
}

func TestFIFO(t *testing.T) {
	server := next.NewNetworkServiceEndpointRegistryServer(
		beginmutex.NewNetworkServiceEndpointRegistryServer(),
		&randomServer{},
		memory.NewNetworkServiceEndpointRegistryServer(),
	)

	count := 10
	nses := []*registry.NetworkServiceEndpoint{}
	for i := 0; i < count; i++ {
		nses = append(nses, &registry.NetworkServiceEndpoint{Name: "nse", Url: fmt.Sprint(i)})
	}

	go server.Register(begin.WithID(context.Background(), 0), nses[0])

	time.Sleep(time.Millisecond * 300)

	var wg sync.WaitGroup
	wg.Add(count - 1)
	for i := 1; i < count; i++ {
		local := i
		go func() {
			_, err := server.Register(begin.WithID(context.Background(), local), nses[local])
			require.NoError(t, err)
			wg.Done()
		}()
		time.Sleep(time.Millisecond * 5)
	}

	wg.Wait()

	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "nse",
		},
	}
	stream, err := adapters.NetworkServiceEndpointServerToClient(server).Find(context.Background(), query)
	require.NoError(t, err)

	nses = registry.ReadNetworkServiceEndpointList(stream)
	require.Len(t, nses, 1)
	require.Equal(t, fmt.Sprint(count-1), nses[0].Url)
}
