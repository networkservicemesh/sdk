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

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

const (
	parallelCount = 1000
)

func TestSerializeServer_StressTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := chain.NewNetworkServiceEndpointRegistryServer(
		begin.NewNetworkServiceEndpointRegistryServer(),
		newParallelServer(t),
	)

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func(id string) {
			defer wg.Done()

			resp, err := server.Register(ctx, &registry.NetworkServiceEndpoint{
				Name: id,
			})
			assert.NoError(t, err)

			_, err = server.Unregister(ctx, resp)
			assert.NoError(t, err)
		}(fmt.Sprint(i % 20))
	}
	wg.Wait()
}

func newParallelServer(t *testing.T) *parallelServer {
	return &parallelServer{
		t: t,
	}
}

type parallelServer struct {
	t      *testing.T
	states sync.Map
}

func (s *parallelServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	raw, _ := s.states.LoadOrStore(in.GetName(), new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "[Register] state has been changed for connection %s expected %d actual %d", in.GetName(), state, atomic.LoadInt32(statePtr))

	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
}

func (s *parallelServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *parallelServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	raw, _ := s.states.LoadOrStore(in.GetName(), new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "[Unregister] state has been changed for connection %s expected %d actual %d", in.GetName(), state, atomic.LoadInt32(statePtr))

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
}
