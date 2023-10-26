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

package beginloop_test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginloop"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type waitGroupServer struct {
	wg *sync.WaitGroup
}

func (s *waitGroupServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.wg.Done()
	log.FromContext(ctx).Infof("Thread [%v] made Done", in.Url)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *waitGroupServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *waitGroupServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

type collectorServer struct {
	mu            sync.Mutex
	registrations []*registry.NetworkServiceEndpoint
}

func (s *collectorServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.mu.Lock()
	s.registrations = append(s.registrations, in)
	s.mu.Unlock()
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *collectorServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *collectorServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

type delayServer struct {
}

func (s *delayServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	milliseconds := rand.Intn(90) + 10
	time.Sleep(time.Millisecond * time.Duration(milliseconds))
	log.FromContext(ctx).Infof("Thread [%v] finished waiting", in.Url)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *delayServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *delayServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

func TestFIFO(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serverWg sync.WaitGroup
	var clientWg sync.WaitGroup
	collector := &collectorServer{}
	server := next.NewNetworkServiceEndpointRegistryServer(
		&waitGroupServer{wg: &serverWg},
		beginloop.NewNetworkServiceEndpointRegistryServer(),
		collector,
		&delayServer{},
	)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	grpcServer := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(grpcServer, server)

	go func() {
		serveErr := grpcServer.Serve(serverLis)
		require.NoError(t, serveErr)
	}()

	clientConn, err := grpc.Dial(serverLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	client := registry.NewNetworkServiceEndpointRegistryClient(clientConn)

	count := 3
	nses := []*registry.NetworkServiceEndpoint{}
	for i := 0; i < count; i++ {
		nses = append(nses, &registry.NetworkServiceEndpoint{Name: "nse", Url: fmt.Sprint(i)})
	}

	clientWg.Add(count)
	for i := 0; i < count; i++ {
		local := i
		serverWg.Add(1)
		go func() {
			_, err := client.Register(begin.WithID(ctx, local), nses[local])
			require.NoError(t, err)
			clientWg.Done()
		}()
		serverWg.Wait()
	}

	clientWg.Wait()

	collector.mu.Lock()
	defer collector.mu.Unlock()
	registrations := collector.registrations

	for i, registration := range registrations {
		require.Equal(t, registration.Url, fmt.Sprint(i))
	}
}
