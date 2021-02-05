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
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

func TestNetworkServiceEndpointRegistryServer_RegisterAndFind(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
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
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
	defer close(ch)
	_ = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))

	require.Equal(t, &registry.NetworkServiceEndpoint{
		Name: "a",
	}, <-ch)
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindWatch(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
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
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
	defer close(ch)
	go func() {
		_ = s.Find(&registry.NetworkServiceEndpointQuery{
			Watch: true,
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: "a",
			},
		}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	}()

	require.Equal(t, &registry.NetworkServiceEndpoint{
		Name: "a",
	}, <-ch)

	expected, err := s.Register(context.Background(), &registry.NetworkServiceEndpoint{
		Name: "a",
	})
	require.NoError(t, err)
	require.True(t, proto.Equal(expected, <-ch))
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindByLabel(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), createLabeledNSE1())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE3())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
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

	require.True(t, proto.Equal(createLabeledNSE2(), <-ch))
}

func TestNetworkServiceEndpointRegistryServer_RegisterAndFindByLabelWatch(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	s := next.NewNetworkServiceEndpointRegistryServer(memory.NewNetworkServiceEndpointRegistryServer())

	_, err := s.Register(context.Background(), createLabeledNSE1())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)

	_, err = s.Register(context.Background(), createLabeledNSE3())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
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

	require.Equal(t, createLabeledNSE2(), <-ch)

	expected, err := s.Register(context.Background(), createLabeledNSE2())
	require.NoError(t, err)
	require.True(t, proto.Equal(expected, <-ch))
}

func TestNetworkServiceEndpointRegistryServer_DataRace(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	go func() {
		for ctx.Err() == nil {
			nse, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
			require.NoError(t, err)

			_, err = s.Unregister(ctx, nse)
			require.NoError(t, err)
		}
	}()

	c := adapters.NetworkServiceEndpointServerToClient(s)

	wg := new(sync.WaitGroup)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse-1"},
				Watch:                  true,
			})
			assert.NoError(t, err)

			for j := 0; j < 100; j++ {
				nse, err := stream.Recv()
				assert.NoError(t, err)

				nse.Name = ""
			}
		}()
	}

	wg.Wait()
}

func TestNetworkServiceEndpointRegistryServer_SlowReceiver(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	c := adapters.NetworkServiceEndpointServerToClient(s)

	findCtx, findCancel := context.WithCancel(ctx)

	stream, err := c.Find(findCtx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "nse"},
		Watch:                  true,
	})
	require.NoError(t, err)

	for i := 0; i < 200; i++ {
		_, err = s.Register(ctx, &registry.NetworkServiceEndpoint{Name: fmt.Sprintf("nse-%d", i)})
		require.NoError(t, err)
	}

	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	_, err = stream.Recv()
	require.NoError(t, err)

	findCancel()
}

func TestNetworkServiceEndpointRegistryServer_ShouldReceiveAllRegisters(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	c := adapters.NetworkServiceEndpointServerToClient(s)

	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		name := fmt.Sprintf("nse-%d", i)

		go func() {
			_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: name})
			require.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithTimeout(ctx, time.Second)
			defer findCancel()

			stream, err := c.Find(findCtx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
				Watch:                  true,
			})
			assert.NoError(t, err)

			_, err = stream.Recv()
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestNetworkServiceEndpointRegistryServer_ShouldReceiveAllUnregisters(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := memory.NewNetworkServiceEndpointRegistryServer()

	for i := 0; i < 200; i++ {
		_, err := s.Register(ctx, &registry.NetworkServiceEndpoint{Name: fmt.Sprintf("nse-%d", i)})
		require.NoError(t, err)
	}

	c := adapters.NetworkServiceEndpointServerToClient(s)

	wg := new(sync.WaitGroup)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		name := fmt.Sprintf("nse-%d", i)

		go func() {
			_, err := s.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: name})
			assert.NoError(t, err)
		}()

		go func() {
			defer wg.Done()

			findCtx, findCancel := context.WithTimeout(ctx, time.Second)
			defer findCancel()

			stream, err := c.Find(findCtx, &registry.NetworkServiceEndpointQuery{
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: name},
				Watch:                  true,
			})
			assert.NoError(t, err)

			exists := false
			for err == nil {
				var nse *registry.NetworkServiceEndpoint
				nse, err = stream.Recv()
				switch {
				case err != nil:
					assert.Equal(t, io.EOF, err)
				case nse.ExpirationTime != nil && nse.ExpirationTime.Seconds < 0:
					return
				default:
					exists = true
				}
			}
			assert.False(t, exists)
		}()
	}
	wg.Wait()
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
