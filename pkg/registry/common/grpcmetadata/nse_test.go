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

func TestGRPCMetadataNetworkServiceEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx := context.Background()

	clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)
	proxyToken, _, _ := tokenGeneratorFunc(proxyID)(nil)

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceEndpointRegistryServer(
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(proxyToken),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(serverID)),
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
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(clientToken),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(proxyID)),
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
		checkcontext.NewNSEClient(t, func(t *testing.T, ctx context.Context) {
			path := grpcmetadata.PathFromContext(ctx)
			require.Equal(t, int(path.Index), 0)
		}),
		grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	_, err = client.Register(ctx, nse)
	require.NoError(t, err)

	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)

	_, err = client.Unregister(ctx, nse)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}

func TestGRPCMetadataNetworkServiceEndpoint_BackwardCompatibility(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cacncel := context.WithTimeout(context.Background(), time.Second)
	defer cacncel()

	clientToken, _, _ := tokenGeneratorFunc(clientID)(nil)

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
		injectpeertoken.NewNetworkServiceEndpointRegistryServer(clientToken),
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(tokenGeneratorFunc(proxyID)),
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
		grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(conn))

	path := grpcmetadata.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	ns := &registry.NetworkServiceEndpoint{Name: "ns"}
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 2)

	_, err = client.Unregister(ctx, ns)
	require.NoError(t, err)
}
