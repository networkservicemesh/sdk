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

package memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

func TestNetworkServiceRegistryServer_RegisterAndFind(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "c",
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkService, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))

	expected := &registry.NetworkService{
		Name: "a",
	}
	require.True(t, proto.Equal(expected, <-ch))
}

func TestNetworkServiceRegistryServer_RegisterAndFindWatch(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	s := next.NewNetworkServiceRegistryServer(memory.NewNetworkServiceRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkService{
		Name: "c",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkService, 1)
	defer close(ch)
	go func() {
		_ = s.Find(&registry.NetworkServiceQuery{
			Watch: true,
			NetworkService: &registry.NetworkService{
				Name: "a",
			},
		}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	}()

	require.Equal(t, &registry.NetworkService{
		Name: "a",
	}, <-ch)

	expected, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(expected, <-ch))
}

func TestNetworkServiceRegistryServer_DataRace(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	go func() {
		for ctx.Err() == nil {
			ns, err := s.Register(ctx, &registry.NetworkService{Name: "ns-1"})
			require.NoError(t, err)

			_, err = s.Unregister(ctx, ns)
			require.NoError(t, err)
		}
	}()

	c := adapters.NetworkServiceServerToClient(s)

	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithTimeout(ctx, time.Second)
			defer findCancel()

			stream, err := c.Find(findCtx, &registry.NetworkServiceQuery{
				NetworkService: &registry.NetworkService{Name: "ns-1"},
				Watch:          true,
			})
			assert.NoError(t, err)

			for j := 0; j < 100; j++ {
				ns, err := stream.Recv()
				assert.NoError(t, err)

				ns.Name = ""
			}
		}()
	}

	wg.Wait()
}

func TestNetworkServiceRegistryServer_SlowReceiver(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	c := adapters.NetworkServiceServerToClient(s)

	findCtx, findCancel := context.WithCancel(ctx)

	stream, err := c.Find(findCtx, &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{Name: "ns-1"},
		Watch:          true,
	})
	require.NoError(t, err)

	for i := 0; i < 100; i++ {
		var ns *registry.NetworkService
		ns, err = s.Register(ctx, &registry.NetworkService{Name: "ns-1"})
		require.NoError(t, err)

		_, err = s.Unregister(ctx, ns)
		require.NoError(t, err)
	}

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	_, err = stream.Recv()
	require.NoError(t, err)

	findCancel()
}

func TestNetworkServiceRegistryServer_ShouldReceiveAllRegisters(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	c := adapters.NetworkServiceServerToClient(s)

	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		name := fmt.Sprintf("ns-%d", i)

		go func() {
			_, err := s.Register(ctx, &registry.NetworkService{Name: name})
			require.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithTimeout(ctx, time.Second)
			defer findCancel()

			stream, err := c.Find(findCtx, &registry.NetworkServiceQuery{
				NetworkService: &registry.NetworkService{Name: name},
				Watch:          true,
			})
			assert.NoError(t, err)

			_, err = stream.Recv()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}
