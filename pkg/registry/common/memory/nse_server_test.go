// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"testing"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
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
