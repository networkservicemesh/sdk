// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package serialize_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serializectx"
)

const (
	parallelCount = 1000
)

func TestSerializeNSEServer_StressTest(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := chain.NewNetworkServiceEndpointRegistryServer(
		serialize.NewNetworkServiceEndpointRegistryServer(),
		new(eventNSEServer),
		newParallelServer(t),
	)

	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func(name string) {
			defer wg.Done()

			reg, err := server.Register(ctx, &registry.NetworkServiceEndpoint{Name: name})
			assert.NoError(t, err)

			_, err = server.Unregister(ctx, reg.Clone())
			assert.NoError(t, err)
		}(fmt.Sprint(i % 20))
	}
	wg.Wait()
}

type eventNSEServer struct{}

func (s *eventNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	executor := serializectx.GetExecutor(ctx, reg.Name)

	go func() {
		executor.AsyncExec(func() {
			registerCtx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			registerCtx = serializectx.WithExecutor(registerCtx, executor)

			_, _ = next.NetworkServiceEndpointRegistryServer(ctx).Register(registerCtx, reg)
		})
	}()

	go func() {
		executor.AsyncExec(func() {
			unregisterCtx, cancel := context.WithCancel(context.TODO())
			defer cancel()

			_, _ = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(unregisterCtx, reg)
		})
	}()

	return reg, nil
}

func (s *eventNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *eventNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

type parallelNSEServer struct {
	t      *testing.T
	states sync.Map
}

func newParallelServer(t *testing.T) *parallelNSEServer {
	return &parallelNSEServer{
		t: t,
	}
}

func (s *parallelNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	reg, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	raw, _ := s.states.LoadOrStore(reg.Name, new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "state has been changed")

	return reg, nil
}

func (s *parallelNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (s *parallelNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	raw, _ := s.states.LoadOrStore(nse.Name, new(int32))
	statePtr := raw.(*int32)

	state := atomic.LoadInt32(statePtr)
	assert.True(s.t, atomic.CompareAndSwapInt32(statePtr, state, state+1), "state has been changed")

	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
