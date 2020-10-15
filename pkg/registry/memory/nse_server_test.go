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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
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
	ch := make(chan *registry.NetworkServiceEndpoint, 1)
	_ = s.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: "a",
		},
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))

	require.Equal(t, &registry.NetworkServiceEndpoint{
		Name: "a",
	}, <-ch)
	cancel()
	close(ch)
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
	require.Equal(t, expected, <-ch)

	close(ch)
}
