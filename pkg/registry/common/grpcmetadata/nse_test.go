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

package grpcmetadata_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectpeertoken"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

type pathCheckerNSEClient struct {
	funcBefore func(ctx context.Context) *grpcmetadata.Path
	funcAfter  func(ctx context.Context, pBefore *grpcmetadata.Path)
}

func newPathCheckerNSEClient(t *testing.T, expectedPathIndex int) registry.NetworkServiceEndpointRegistryClient {
	client := &pathCheckerNSEClient{}

	client.funcBefore = func(ctx context.Context) *grpcmetadata.Path {
		p := grpcmetadata.PathFromContext(ctx).Clone()
		require.Equal(t, int(p.Index), expectedPathIndex)

		return p
	}
	client.funcAfter = func(ctx context.Context, pBefore *grpcmetadata.Path) {
		pAfter := grpcmetadata.PathFromContext(ctx).Clone()
		require.Equal(t, int(pAfter.Index), expectedPathIndex)
		for i := expectedPathIndex; i < len(pBefore.PathSegments); i++ {
			require.NotEqual(t, pBefore.PathSegments[i].Token, pAfter.PathSegments[i].Token)
		}
	}
	return client
}

func (p *pathCheckerNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	pBefore := p.funcBefore(ctx)
	r, e := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
	p.funcAfter(ctx, pBefore)
	return r, e
}

func (p *pathCheckerNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	pBefore := p.funcBefore(ctx)
	c, e := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	p.funcAfter(ctx, pBefore)

	return c, e
}

func (p *pathCheckerNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	pBefore := p.funcBefore(ctx)
	r, e := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
	p.funcAfter(ctx, pBefore)
	return r, e
}

// nolint: funlen
// This test checks that registry Path is correctly updated and passed through grpc metadata
// Test scheme: client ---> proxyServer ---> server
func TestGRPCMetadataNetworkServiceEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Add clockMock to the context
	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	// tokenGeneratorFunc generates new tokens automatically
	server := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, serverID)),
		checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 2)
		}),
	)

	serverGRPCServer := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(serverGRPCServer, server)
	go func() {
		serveErr := serverGRPCServer.Serve(serverLis)
		require.NoError(t, serveErr)
	}()

	proxyLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverConn, err := grpc.Dial(serverLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := serverConn.Close()
		require.NoError(t, closeErr)
	}()

	proxyServer := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, clientID)),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 1)
		}),
		adapters.NetworkServiceEndpointClientToServer(next.NewNetworkServiceEndpointRegistryClient(
			newPathCheckerNSEClient(t, 1),
			grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
			registry.NewNetworkServiceEndpointRegistryClient(serverConn),
		)),
	)

	proxyGRPCServer := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(proxyGRPCServer, proxyServer)
	go func() {
		serveErr := proxyGRPCServer.Serve(proxyLis)
		require.NoError(t, serveErr)
	}()

	conn, err := grpc.Dial(proxyLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := conn.Close()
		require.NoError(t, closeErr)
	}()

	client := next.NewNetworkServiceEndpointRegistryClient(
		newPathCheckerNSEClient(t, 0),
		grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	nse, err = client.Register(ctx, nse)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)
	require.Len(t, nse.PathIds, 3)

	// Simulate refresh
	_, err = client.Register(ctx, nse)
	require.NoError(t, err)

	query := &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: nse}
	path = grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)
	_, err = client.Find(ctx, query)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)
	//require.Len(t, query.NetworkService.PathIds, 3)

	_, err = client.Unregister(ctx, nse)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}

// nolint: funlen
func TestGRPCMetadataNetworkServiceEndpoint_BackwardCompatibility(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Add clockMock to the context
	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceEndpointRegistryServer()

	serverGRPCServer := grpc.NewServer()
	defer func() {
		serverGRPCServer.Stop()
	}()
	registry.RegisterNetworkServiceEndpointRegistryServer(serverGRPCServer, server)
	go func() {
		serveErr := serverGRPCServer.Serve(serverLis)
		require.NoError(t, serveErr)
	}()

	proxyLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	serverConn, err := grpc.Dial(serverLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := serverConn.Close()
		require.NoError(t, closeErr)
	}()

	proxyServer := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, clientID)),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 1)
		}),
		adapters.NetworkServiceEndpointClientToServer(next.NewNetworkServiceEndpointRegistryClient(
			grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
			registry.NewNetworkServiceEndpointRegistryClient(serverConn),
		)),
	)

	proxyGRPCServer := grpc.NewServer()
	defer func() {
		proxyGRPCServer.Stop()
	}()
	registry.RegisterNetworkServiceEndpointRegistryServer(proxyGRPCServer, proxyServer)
	go func() {
		serveErr := proxyGRPCServer.Serve(proxyLis)
		require.NoError(t, serveErr)
	}()

	conn, err := grpc.Dial(proxyLis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := conn.Close()
		require.NoError(t, closeErr)
	}()

	client := next.NewNetworkServiceEndpointRegistryClient(
		newPathCheckerNSEClient(t, 0),
		grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	nse := &registry.NetworkServiceEndpoint{Name: "ns"}
	nse, err = client.Register(ctx, nse)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)
	require.Len(t, nse.PathIds, 2)

	query := &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: nse}
	path = grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)
	_, err = client.Find(ctx, query)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)
	//require.Len(t, query.NetworkService.PathIds, 3)

	// Simulate refresh
	_, err = client.Register(ctx, nse)
	require.NoError(t, err)

	_, err = client.Unregister(ctx, nse)
	require.NoError(t, err)
}
