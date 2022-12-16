// Copyright (c) 2022 Cisco and/or its affiliates.
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/inject/injectpeertoken"

	"go.uber.org/goleak"
)

const (
	clientID = "spiffe://test.com/client"
	proxyID  = "spiffe://test.com/proxy"
	serverID = "spiffe://test.com/server"
)

func TestGRPCMetadataNetworkService(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cacncel := context.WithTimeout(context.Background(), time.Second)
	defer cacncel()

	clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)
	proxyToken, _, _ := tokenGeneratorFunc(proxyID)(nil)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceRegistryServer(
		injectpeertoken.NewNetworkServiceRegistryServer(proxyToken),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(serverID)),
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
		injectpeertoken.NewNetworkServiceRegistryServer(clientToken),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
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
		grpcmetadata.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	ns := &registry.NetworkService{Name: "ns"}
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)

	_, err = client.Unregister(ctx, ns)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}

func TestGRPCMetadataNetworkService_BackwardCompatibility(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cacncel := context.WithTimeout(context.Background(), time.Second)
	defer cacncel()

	clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)

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
		injectpeertoken.NewNetworkServiceRegistryServer(clientToken),
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer(tokenGeneratorFunc(proxyID)),
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
		grpcmetadata.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	ns := &registry.NetworkService{Name: "ns"}
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)

	_, err = client.Unregister(ctx, ns)
	require.NoError(t, err)
}
