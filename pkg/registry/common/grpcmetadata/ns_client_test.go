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

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/registry/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/token"

	"go.uber.org/goleak"
)

func tokenGenerator(spiffeID string) token.GeneratorFunc {
	return func(authInfo credentials.AuthInfo) (string, time.Time, error) {
		expireTime := time.Now().Add(time.Hour)
		claims := jwt.RegisteredClaims{
			Subject:   spiffeID,
			ExpiresAt: jwt.NewNumericDate(expireTime),
		}

		tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("supersecret"))
		return tok, expireTime, err
	}
}

type testTokenNSServer struct {
	tokenGenerator token.GeneratorFunc
}

func NewNetworkServiceRegistryClient(tokenGenerator token.GeneratorFunc) registry.NetworkServiceRegistryClient {
	return &testTokenNSServer{
		tokenGenerator: tokenGenerator,
	}
}

func (s *testTokenNSServer) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	tok, expire, _ := s.tokenGenerator(nil)
	ctx = metadata.AppendToOutgoingContext(ctx, "nsm-client-token", tok, "nsm-client-token-expires", expire.Format(time.RFC3339Nano))

	return next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
}

func (s *testTokenNSServer) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
}

func (s *testTokenNSServer) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}

func TestAuthzNetworkServiceRegistry(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx := context.Background()

	serverAddress := "localhost:44000"
	serverLis, err := net.Listen("tcp", serverAddress)
	require.NoError(t, err)

	server := next.NewNetworkServiceRegistryServer(
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer("server"),
		updatetoken.NewNetworkServiceRegistryServer(tokenGenerator("spiffe://test.com/server-token")),
	)

	serverGRPCServer := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(serverGRPCServer, server)
	go func() {
		serveErr := serverGRPCServer.Serve(serverLis)
		require.NoError(t, serveErr)
	}()

	proxyAddress := "localhost:45000"
	proxyLis, err := net.Listen("tcp", proxyAddress)
	require.NoError(t, err)

	serverConn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := serverConn.Close()
		require.NoError(t, closeErr)
	}()

	proxyServer := next.NewNetworkServiceRegistryServer(
		grpcmetadata.NewNetworkServiceRegistryServer(),
		updatepath.NewNetworkServiceRegistryServer("proxy-server"),
		updatetoken.NewNetworkServiceRegistryServer(tokenGenerator("spiffe://test.com/proxy-server-token")),
		adapters.NetworkServiceClientToServer(NewNetworkServiceRegistryClient(tokenGenerator("spiffe://test.com/client-token"))),
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

	conn, err := grpc.Dial(proxyAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		closeErr := conn.Close()
		require.NoError(t, closeErr)
	}()

	client := next.NewNetworkServiceRegistryClient(
		NewNetworkServiceRegistryClient(tokenGenerator("spiffe://test.com/client-token")),
		updatepath.NewNetworkServiceRegistryClient("client"),
		grpcmetadata.NewNetworkServiceRegistryClient(),
		registry.NewNetworkServiceRegistryClient(conn))

	ns := &registry.NetworkService{Name: "ns"}
	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	_, err = client.Register(ctx, ns)
	require.NoError(t, err)

	serverGRPCServer.Stop()
	proxyGRPCServer.Stop()
}
