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

package checkid_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/networkservicemesh/sdk/pkg/registry/common/checkid"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

const (
	nseName      = "nse"
	nseURL       = "tcp://0.0.0.0"
	duplicateURL = "tcp://1.1.1.1"
)

func testNSE(u string) *registry.NetworkServiceEndpoint {
	return &registry.NetworkServiceEndpoint{
		Name: nseName,
		Url:  u,
	}
}

func TestCheckIDServer_Register(t *testing.T) {
	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		checkid.NewNetworkServiceEndpointRegistryServer(),
		mem,
	)

	// 1. Register
	reg, err := s.Register(context.Background(), testNSE(nseURL))
	require.NoError(t, err)
	require.Equal(t, nseName, reg.Name)
	require.Equal(t, nseURL, reg.Url)

	nses := find(t, mem, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})

	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].Name)
	require.Equal(t, nseURL, nses[0].Url)

	// 2. Refresh
	reg, err = s.Register(context.Background(), reg.Clone())
	require.NoError(t, err)
	require.Equal(t, nseName, reg.Name)
	require.Equal(t, nseURL, reg.Url)

	nses = find(t, mem, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})

	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].Name)
	require.Equal(t, nseURL, nses[0].Url)

	// 3. Unregister
	_, err = s.Unregister(context.Background(), reg.Clone())
	require.NoError(t, err)

	nses = find(t, mem, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})

	require.Empty(t, nses)

	// 4. Register duplicate
	reg, err = s.Register(context.Background(), testNSE(duplicateURL))
	require.NoError(t, err)
	require.Equal(t, nseName, reg.Name)
	require.Equal(t, duplicateURL, reg.Url)

	nses = find(t, mem, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})

	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].Name)
	require.Equal(t, duplicateURL, nses[0].Url)
}

func TestCheckIDServer_Duplicate(t *testing.T) {
	mem := memory.NewNetworkServiceEndpointRegistryServer()

	s := next.NewNetworkServiceEndpointRegistryServer(
		checkid.NewNetworkServiceEndpointRegistryServer(),
		mem,
	)

	// 1. Register
	reg, err := s.Register(context.Background(), testNSE(nseURL))
	require.NoError(t, err)
	require.Equal(t, nseName, reg.Name)
	require.Equal(t, nseURL, reg.Url)

	// 2. Register duplicate
	_, err = s.Register(context.Background(), testNSE(duplicateURL))
	require.Error(t, err)

	grpcStatus, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.AlreadyExists, grpcStatus.Code())

	nses := find(t, mem, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})

	require.Len(t, nses, 1)
	require.Equal(t, nseName, nses[0].Name)
	require.Equal(t, nseURL, nses[0].Url)
}

func TestCheckIDServer_RemoteDuplicate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(server, checkid.NewNetworkServiceEndpointRegistryServer())

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}

	errCh := grpcutils.ListenAndServe(ctx, u, server)
	select {
	case err := <-errCh:
		require.NoError(t, err)
	default:
	}

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(u), grpc.WithInsecure())
	require.NoError(t, err)

	c := registry.NewNetworkServiceEndpointRegistryClient(cc)

	// 1. Register
	_, err = c.Register(context.Background(), testNSE(nseURL))
	require.NoError(t, err)

	// 2. Register duplicate
	_, err = c.Register(context.Background(), testNSE(duplicateURL))
	require.Error(t, err)

	grpcStatus, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.AlreadyExists, grpcStatus.Code())
}

func find(t *testing.T, mem registry.NetworkServiceEndpointRegistryServer, query *registry.NetworkServiceEndpointQuery) (nses []*registry.NetworkServiceEndpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch := make(chan *registry.NetworkServiceEndpoint)
	go func() {
		defer close(ch)
		require.NoError(t, mem.Find(query, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch)))
	}()

	for nse := range ch {
		nses = append(nses, nse)
	}

	return nses
}
