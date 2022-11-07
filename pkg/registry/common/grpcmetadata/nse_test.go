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

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/utils/checks/checkcontext"

	"go.uber.org/goleak"
)

func TestGRPCMetadataNetworkServiceEndpoint(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx := context.Background()

	serverLis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := next.NewNetworkServiceEndpointRegistryServer(
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(serverName),
		checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
			path, checkErr := grpcmetadata.PathFromContext(ctx)
			require.NoError(t, checkErr)

			require.Equal(t, int(path.Index), 2)
			require.Equal(t, path.PathSegments[0].Name, clientName)
			require.Equal(t, path.PathSegments[1].Name, proxyName)
			require.Equal(t, path.PathSegments[2].Name, serverName)
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
		grpcmetadata.NewNetworkServiceEndpointRegistryServer(),
		updatepath.NewNetworkServiceEndpointRegistryServer(proxyName),
		checkcontext.NewNSEServer(t, func(t *testing.T, ctx context.Context) {
			path, checkErr := grpcmetadata.PathFromContext(ctx)
			require.NoError(t, checkErr)

			require.Equal(t, int(path.Index), 1)
			require.Equal(t, path.PathSegments[0].Name, clientName)
			require.Equal(t, path.PathSegments[1].Name, proxyName)
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
		updatepath.NewNetworkServiceEndpointRegistryClient(clientName),
		checkcontext.NewNSEClient(t, func(t *testing.T, ctx context.Context) {
			path, checkErr := grpcmetadata.PathFromContext(ctx)
			require.NoError(t, checkErr)

			require.Equal(t, int(path.Index), 0)
			require.Equal(t, path.PathSegments[0].Name, clientName)
		}),
		grpcmetadata.NewNetworkServiceEndpointRegistryClient(),
		registry.NewNetworkServiceEndpointRegistryClient(conn))

	path := registry.Path{}
	ctx = grpcmetadata.PathWithContext(ctx, &path)

	nse := &registry.NetworkServiceEndpoint{Name: "nse"}
	_, err = client.Register(ctx, nse)
	require.NoError(t, err)

	require.Equal(t, int(path.Index), 0)
	require.Len(t, path.PathSegments, 3)
	require.Equal(t, path.PathSegments[0].Name, clientName)
	require.Equal(t, path.PathSegments[1].Name, proxyName)
	require.Equal(t, path.PathSegments[2].Name, serverName)

	_, err = client.Unregister(ctx, nse)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}
