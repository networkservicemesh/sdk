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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

func TestNetworkServiceEndpointRegistryServer_RegisterAndFind(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "c",
	})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	isResponseEqual := proto.Equal(<-ch, &registry.NetworkServiceEndpointResponse{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	})
	require.True(t, isResponseEqual)
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindWatch(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "b",
	})
	require.NoError(t, err)

	_, err = s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "c",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	go func() {
		_ = s.Find(&registry.NetworkServiceEndpointQuery{
			Watch: true,
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: "a",
			},
		}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	}()
	isResponseEqual := proto.Equal(<-ch, &registry.NetworkServiceEndpointResponse{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	})
	require.True(t, isResponseEqual)

	expected, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(&registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: expected}, <-ch))
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindByLabel(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), createLabeledNSE1())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE3())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
				"Service1": {
					Labels: map[string]string{
						"c": "d",
					},
				},
			},
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))

	require.True(t, proto.Equal(&registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: createLabeledNSE2()}, <-ch))
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindByLabelWatch(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), createLabeledNSE1())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE3())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	defer close(ch)
	go func() {
		_ = s.Find(&registry.NetworkServiceEndpointQuery{
			Watch: true,
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				NetworkServiceLabels: map[string]*registry.NetworkServiceLabels{
					"Service1": {
						Labels: map[string]string{
							"c": "d",
						},
					},
				},
			},
		}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	}()

	isResponseEqual := proto.Equal(<-ch, &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: createLabeledNSE2()})
	require.True(t, isResponseEqual)

	expected, err := s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)
	require.True(t, proto.Equal(&registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: expected}, <-ch))
}

func TestNetworkServiceEndpointRegistryServer_DataRace(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse"})
	require.NoError(t, err)

	var wgStart, wgEnd sync.WaitGroup
	for i := 0; i < 10; i++ {
		wgStart.Add(1)
		wgEnd.Add(1)
		go func() {
			defer wgEnd.Done()

			findCtx, findCancel := context.WithCancel(ctx)
			defer findCancel()

			ch := make(chan *registry.NetworkServiceEndpointResponse, 10)
			go func() {
				defer close(ch)
				findErr := s.Find(&registry.NetworkServiceEndpointQuery{
					NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
					Watch:                  true,
				}, streamchannel.NewNetworkServiceEndpointFindServer(findCtx, ch))
				assert.NoError(t, findErr)
			}()

			_, receiveErr := receiveNSER(findCtx, ch)
			assert.NoError(t, receiveErr)

			wgStart.Done()

			for j := 0; j < 50; j++ {
				nse, receiveErr := receiveNSER(findCtx, ch)
				assert.NoError(t, receiveErr)

				nse.NetworkServiceEndpoint.Name = ""
			}
		}()
	}
	wgWait(ctx, t, &wgStart)

	for i := 0; i < 50; i++ {
		_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: fmt.Sprintf("nse-%d", i)})
		require.NoError(t, err)
	}

	wgWait(ctx, t, &wgEnd)
}

func TestNetworkServiceEndpointRegistryServer_SlowReceiver(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	findCtx, findCancel := context.WithCancel(ctx)

	ch := make(chan *registry.NetworkServiceEndpointResponse, 10)
	go func() {
		defer close(ch)
		findErr := s.Find(&registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
			Watch:                  true,
		}, streamchannel.NewNetworkServiceEndpointFindServer(findCtx, ch))
		require.NoError(t, findErr)
	}()

	for i := 0; i < 50; i++ {
		_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: fmt.Sprintf("nse-%d", i)})
		require.NoError(t, err)
	}

	ignoreCurrent := goleak.IgnoreCurrent()

	_, err := receiveNSER(findCtx, ch)
	require.NoError(t, err)

	findCancel()

	require.Eventually(t, func() bool {
		return goleak.Find(ignoreCurrent) == nil
	}, 100*time.Millisecond, time.Millisecond)
}

func TestNetworkServiceEndpointRegistryServer_ShouldReceiveAllRegisters(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		name := fmt.Sprintf("nse-%d", i)

		go func() {
			_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: name})
			require.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithCancel(ctx)
			defer findCancel()

			ch := make(chan *registry.NetworkServiceEndpointResponse, 10)
			go func() {
				defer close(ch)
				err := s.Find(&registry.NetworkServiceEndpointQuery{
					NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
					Watch:                  true,
				}, streamchannel.NewNetworkServiceEndpointFindServer(findCtx, ch))
				assert.NoError(t, err)
			}()

			_, err := receiveNSER(findCtx, ch)
			assert.NoError(t, err)
		}()
	}
	wgWait(ctx, t, &wg)
}

func TestNetworkServiceEndpointRegistryServer_ShouldReceiveAllUnregisters(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	for i := 0; i < 50; i++ {
		_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: fmt.Sprintf("nse-%d", i)})
		require.NoError(t, err)
	}

	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("nse-%d", i)

		go func() {
			_, err := s.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: name})
			assert.NoError(t, err)
		}()

		go func() {
			findCtx, findCancel := context.WithCancel(ctx)
			defer findCancel()

			ch := make(chan *registry.NetworkServiceEndpointResponse, 10)
			go func() {
				defer close(ch)
				err := s.Find(&registry.NetworkServiceEndpointQuery{
					NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
					Watch:                  true,
				}, streamchannel.NewNetworkServiceEndpointFindServer(findCtx, ch))
				assert.NoError(t, err)
			}()

			var err error
			exists := false
			for err == nil {
				var nseResp *registry.NetworkServiceEndpointResponse
				nseResp, err = receiveNSER(findCtx, ch)
				switch {
				case err != nil:
					assert.Equal(t, io.EOF, err)
				case nseResp.GetDeleted():
					return
				default:
					exists = true
				}
			}
			assert.False(t, exists)
		}()
	}
	<-ctx.Done()
}

func createLabeledNSE1() *registry.NetworkServiceEndpoint {
	labels := map[string]*registry.NetworkServiceLabels{
		"Service1": {
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}
	return &registry.NetworkServiceEndpoint{
		Name: "nse1",
		NetworkServiceNames: []string{
			"Service1",
		},
		NetworkServiceLabels: labels,
	}
}

func createLabeledNSE2() *registry.NetworkServiceEndpoint {
	labels := map[string]*registry.NetworkServiceLabels{
		"Service1": {
			Labels: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
		"Service2": {
			Labels: map[string]string{
				"1": "2",
				"3": "4",
			},
		},
	}
	return &registry.NetworkServiceEndpoint{
		Name: "nse2",
		NetworkServiceNames: []string{
			"Service1", "Service2",
		},
		NetworkServiceLabels: labels,
	}
}

func createLabeledNSE3() *registry.NetworkServiceEndpoint {
	labels := map[string]*registry.NetworkServiceLabels{
		"Service555": {
			Labels: map[string]string{
				"a": "b",
				"c": "d",
			},
		},
	}
	return &registry.NetworkServiceEndpoint{
		Name: "nse3",
		NetworkServiceNames: []string{
			"Service1",
		},
		NetworkServiceLabels: labels,
	}
}

func receiveNSER(ctx context.Context, ch <-chan *registry.NetworkServiceEndpointResponse) (*registry.NetworkServiceEndpointResponse, error) {
	select {
	case <-ctx.Done():
		return nil, io.EOF
	case nseResp, ok := <-ch:
		if !ok {
			return nil, io.EOF
		}
		return nseResp, nil
	}
}
