// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco Systems, Inc.
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
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

func TestNetworkServiceRegistryServer_RegisterAndFind(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
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
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))

	expected := &registry.NetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	}

	require.True(t, proto.Equal(expected, <-ch))
}

func TestNetworkServiceRegistryServer_RegisterAndFindWatch(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
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
	ch := make(chan *registry.NetworkServiceResponse, 1)
	defer close(ch)
	go func() {
		_ = s.Find(&registry.NetworkServiceQuery{
			Watch: true,
			NetworkService: &registry.NetworkService{
				Name: "a",
			},
		}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	}()

	isResponseEqual := proto.Equal(<-ch, &registry.NetworkServiceResponse{
		NetworkService: &registry.NetworkService{
			Name: "a",
		},
	})
	require.True(t, isResponseEqual)
	expected, err := s.Register(context.Background(), &registry.NetworkService{
		Name: "a",
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(&registry.NetworkServiceResponse{NetworkService: expected}, <-ch))
}

func TestNetworkServiceRegistryServer_DataRace(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	_, err := s.Register(ctx, &registry.NetworkService{Name: "ns"})
	require.NoError(t, err)

	var wgStart, wgEnd sync.WaitGroup
	for i := 0; i < 10; i++ {
		wgStart.Add(1)
		wgEnd.Add(1)
		go func() {
			defer wgEnd.Done()

			findCtx, findCancel := context.WithCancel(ctx)
			defer findCancel()

			ch := make(chan *registry.NetworkServiceResponse, 10)
			go func() {
				defer close(ch)
				findErr := s.Find(&registry.NetworkServiceQuery{
					NetworkService: &registry.NetworkService{},
					Watch:          true,
				}, streamchannel.NewNetworkServiceFindServer(findCtx, ch))
				assert.NoError(t, findErr)
			}()

			_, receiveErr := readNSResponse(findCtx, ch)
			assert.NoError(t, receiveErr)

			wgStart.Done()

			for j := 0; j < 50; j++ {
				nseResp, receiveErr := readNSResponse(findCtx, ch)
				assert.NoError(t, receiveErr)

				nseResp.NetworkService.Name = ""
			}
		}()
	}
	wgWait(ctx, t, &wgStart)

	for i := 0; i < 50; i++ {
		_, err := s.Register(ctx, &registry.NetworkService{Name: fmt.Sprintf("ns-%d", i)})
		require.NoError(t, err)
	}

	wgWait(ctx, t, &wgEnd)
}

func TestNetworkServiceRegistryServer_SlowReceiver(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	findCtx, findCancel := context.WithCancel(ctx)

	ch := make(chan *registry.NetworkServiceResponse, 10)
	go func() {
		defer close(ch)
		findErr := s.Find(&registry.NetworkServiceQuery{
			NetworkService: &registry.NetworkService{},
			Watch:          true,
		}, streamchannel.NewNetworkServiceFindServer(findCtx, ch))
		require.NoError(t, findErr)
	}()

	for i := 0; i < 50; i++ {
		_, err := s.Register(ctx, &registry.NetworkService{Name: fmt.Sprintf("ns-%d", i)})
		require.NoError(t, err)
	}

	ignoreCurrent := goleak.IgnoreCurrent()

	_, err := readNSResponse(findCtx, ch)
	require.NoError(t, err)

	findCancel()

	require.Eventually(t, func() bool {
		return goleak.Find(ignoreCurrent) == nil
	}, 100*time.Millisecond, time.Millisecond)
}

func TestNetworkServiceRegistryServer_ShouldReceiveAllRegisters(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceRegistryServer()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		name := fmt.Sprintf("ns-%d", i)

		go func() {
			_, err := s.Register(ctx, &registry.NetworkService{Name: name})
			require.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithCancel(ctx)
			defer findCancel()

			ch := make(chan *registry.NetworkServiceResponse, 10)
			go func() {
				defer close(ch)
				err := s.Find(&registry.NetworkServiceQuery{
					NetworkService: &registry.NetworkService{Name: name},
					Watch:          true,
				}, streamchannel.NewNetworkServiceFindServer(findCtx, ch))
				assert.NoError(t, err)
			}()

			_, err := readNSResponse(findCtx, ch)
			assert.NoError(t, err)
		}()
	}
	wgWait(ctx, t, &wg)
}

func readNSResponse(ctx context.Context, ch <-chan *registry.NetworkServiceResponse) (*registry.NetworkServiceResponse, error) {
	select {
	case <-ctx.Done():
		return nil, io.EOF
	case nsResp, ok := <-ch:
		if !ok {
			return nil, io.EOF
		}
		return nsResp, nil
	}
}
