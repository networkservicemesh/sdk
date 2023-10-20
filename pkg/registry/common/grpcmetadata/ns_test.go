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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectpeertoken"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	clientID = "spiffe://test.com/client"
	proxyID  = "spiffe://test.com/proxy"
	serverID = "spiffe://test.com/server"
)

type pathCheckerNSClient struct {
	funcBefore func(ctx context.Context) *grpcmetadata.Path
	funcAfter  func(ctx context.Context, pBefore *grpcmetadata.Path)
}

func newPathCheckerNSClient(t *testing.T, expectedPathIndex int) registry.NetworkServiceRegistryClient {
	client := &pathCheckerNSClient{}

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

func (p *pathCheckerNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	pBefore := p.funcBefore(ctx)
	r, e := next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
	p.funcAfter(ctx, pBefore)
	return r, e
}

func (p *pathCheckerNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	pBefore := p.funcBefore(ctx)
	c, e := next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	p.funcAfter(ctx, pBefore)
	return c, e
}

func (p *pathCheckerNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	pBefore := p.funcBefore(ctx)
	r, e := next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
	p.funcAfter(ctx, pBefore)
	return r, e
}

// nolint: funlen
func TestGRPCMetadataNetworkService(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2000)
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceRegistryServer(
		injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, serverID)),
		checkcontext.NewNSServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 2)
		}),
	)

	serverGRPCServer := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(serverGRPCServer, server)
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

	proxyServer := next.NewNetworkServiceRegistryServer(
		injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, clientID)),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		checkcontext.NewNSServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 1)
		}),
		adapters.NetworkServiceClientToServer(next.NewNetworkServiceRegistryClient(
			newPathCheckerNSClient(t, 1),
			grpcmetadata.NewNetworkServiceRegistryClient(),
			registry.NewNetworkServiceRegistryClient(serverConn),
		)),
	)

	proxyGRPCServer := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(proxyGRPCServer, proxyServer)
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

	client := next.NewNetworkServiceRegistryClient(
		newPathCheckerNSClient(t, 0),
		grpcmetadata.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	ns := &registry.NetworkService{Name: "ns"}
	ns, err = client.Register(ctx, ns)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)
	require.Len(t, ns.PathIds, 3)

	// Simulate refresh
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	query := &registry.NetworkServiceQuery{NetworkService: ns}
	path = grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)
	_, err = client.Find(ctx, query)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)
	//require.Len(t, query.NetworkService.PathIds, 3)

	_, err = client.Unregister(ctx, ns)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}

// nolint: funlen
func TestGRPCMetadataNetworkService_BackwardCompatibility(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceRegistryServer()

	serverGRPCServer := grpc.NewServer()
	defer func() {
		serverGRPCServer.Stop()
	}()
	registry.RegisterNetworkServiceRegistryServer(serverGRPCServer, server)
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

	proxyServer := next.NewNetworkServiceRegistryServer(
		injectpeertoken.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, clientID)),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(clockMock, proxyID)),
		checkcontext.NewNSServer(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 1)
		}),
		adapters.NetworkServiceClientToServer(next.NewNetworkServiceRegistryClient(
			grpcmetadata.NewNetworkServiceRegistryClient(),
			registry.NewNetworkServiceRegistryClient(serverConn),
		)),
	)

	proxyGRPCServer := grpc.NewServer()
	defer func() {
		proxyGRPCServer.Stop()
	}()
	registry.RegisterNetworkServiceRegistryServer(proxyGRPCServer, proxyServer)
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

	client := next.NewNetworkServiceRegistryClient(
		newPathCheckerNSClient(t, 0),
		grpcmetadata.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	ns := &registry.NetworkService{Name: "ns"}
	ns, err = client.Register(ctx, ns)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)
	require.Len(t, ns.PathIds, 2)

	// Simulate refresh
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	query := &registry.NetworkServiceQuery{NetworkService: ns}
	path = grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)
	_, err = client.Find(ctx, query)
	require.NoError(t, err)
	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)
	//require.Len(t, query.NetworkService.PathIds, 2)

	_, err = client.Unregister(ctx, ns)
	require.NoError(t, err)
}
