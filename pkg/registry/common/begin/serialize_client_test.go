// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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
	"sync/atomic"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func TestSerializeClient_StressTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := chain.NewNetworkServiceEndpointRegistryClient(
		begin.NewNetworkServiceEndpointRegistryClient(),
		newParallelClient(t),
	)

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func(id string) {
			defer wg.Done()

			resp, err := client.Register(ctx, &registry.NetworkServiceEndpoint{
				Name: id,
			})
			assert.NoError(t, err)

			_, err = client.Unregister(ctx, resp)
			assert.NoError(t, err)
		}(fmt.Sprint(i % 20))
	}
	wg.Wait()
}

type parallelClient struct {
	t      *testing.T
	states sync.Map
	mu     sync.Mutex
}

func newParallelClient(t *testing.T) *parallelClient {
	return &parallelClient{
		t: t,
	}
}

func (s *parallelClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	raw, _ := s.states.LoadOrStore(in.GetName(), new(int32))
	statePtr := raw.(*int32)

	s.mu.Lock()
	state := atomic.LoadInt32(statePtr)
	if !atomic.CompareAndSwapInt32(statePtr, state, state+1) {
		assert.Failf(s.t, "", "state has been changed for connection %s expected %d actual %d", in.GetName(), state, atomic.LoadInt32(statePtr))
	}
	s.mu.Unlock()

	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (s *parallelClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (s *parallelClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	raw, _ := s.states.LoadOrStore(in.GetName(), new(int32))
	statePtr := raw.(*int32)

	s.mu.Lock()
	state := atomic.LoadInt32(statePtr)
	if !atomic.CompareAndSwapInt32(statePtr, state, state+1) {
		assert.Failf(s.t, "", "state has been changed for connection %s expected %d actual %d", in.GetName(), state, atomic.LoadInt32(statePtr))
	}
	s.mu.Unlock()

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceEndpointRegistryClient - returns a new null client that does nothing but call next.NetworkServiceEndpointRegistryClient(ctx).
func NewNetworkServiceEndpointRegistryClient() registry.NetworkServiceEndpointRegistryClient {
	return new(parallelClient)
}
