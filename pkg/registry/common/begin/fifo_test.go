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
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginloop"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginmutex"
	"github.com/networkservicemesh/sdk/pkg/registry/common/beginrecursive2"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestFIFOBegin(t *testing.T) {
	testFIFO(t, begin.NewNetworkServiceEndpointRegistryServer())
}

func TestFIFOBeginChannels(t *testing.T) {
	testFIFO(t, beginloop.NewNetworkServiceEndpointRegistryServer())
}

func TestFIFOBeginMutex(t *testing.T) {
	testFIFO(t, beginmutex.NewNetworkServiceEndpointRegistryServer())
}

func TestFIFOBeginExecutor(t *testing.T) {
	testFIFO(t, beginrecursive2.NewNetworkServiceEndpointRegistryServer())
}

func testFIFO(t *testing.T, beginServer registry.NetworkServiceEndpointRegistryServer) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var serverWg sync.WaitGroup
	var clientWg sync.WaitGroup
	collector := &collectorServer{}
	server := next.NewNetworkServiceEndpointRegistryServer(
		&waitGroupServer{wg: &serverWg},
		beginServer,
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

	count := 1000
	nses := []*registry.NetworkServiceEndpoint{}
	for i := 0; i < count; i++ {
		nses = append(nses, &registry.NetworkServiceEndpoint{Name: "nse", Url: fmt.Sprint(i)})
	}

	expected := make([]request, 0)

	clientWg.Add(count)
	for i := 0; i < count; i++ {
		local := i
		serverWg.Add(1)
		go func() {
			var err error
			if local%2 == 0 {
				expected = append(expected, request{requestType: register, requestData: nses[local]})
				_, err = client.Register(begin.WithID(ctx, local), nses[local])
			} else {
				expected = append(expected, request{requestType: unregister, requestData: nses[local]})
				_, err = client.Unregister(begin.WithID(ctx, local), nses[local])
			}
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
		require.Equal(t, registration.requestData.Url, expected[i].requestData.Url)
		require.Equal(t, registration.requestType, expected[i].requestType)
	}
}

type eventType int

const (
	register   eventType = 0
	unregister eventType = 1
)

type request struct {
	requestType eventType
	requestData *registry.NetworkServiceEndpoint
}

type waitGroupServer struct {
	wg *sync.WaitGroup
}

func (s *waitGroupServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.wg.Done()
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *waitGroupServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *waitGroupServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.wg.Done()
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

type collectorServer struct {
	mu            sync.Mutex
	registrations []request
}

func (s *collectorServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	s.mu.Lock()
	s.registrations = append(s.registrations, request{
		requestType: register,
		requestData: in,
	})
	s.mu.Unlock()
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *collectorServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *collectorServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	s.mu.Lock()
	s.registrations = append(s.registrations, request{
		requestType: unregister,
		requestData: in,
	})
	s.mu.Unlock()
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}

type delayServer struct {
}

func (s *delayServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	milliseconds := rand.Intn(90) + 10
	time.Sleep(time.Millisecond * time.Duration(milliseconds))
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *delayServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *delayServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	milliseconds := rand.Intn(90) + 10
	time.Sleep(time.Millisecond * time.Duration(milliseconds))
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}
