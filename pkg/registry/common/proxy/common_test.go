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

package proxy_test

import (
	"context"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/proxy"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startNSEServer(t *testing.T, chain registry.NetworkServiceEndpointRegistryServer) (u *url.URL, closeFunc func()) {
	s := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(s, chain)
	grpcutils.RegisterHealthServices(s, chain)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	closeFunc = func() {
		_ = l.Close()
	}
	go func() {
		_ = s.Serve(l)
	}()
	u, err = url.Parse("tcp://" + l.Addr().String())
	if err != nil {
		closeFunc()
	}
	require.Nil(t, err)
	return u, closeFunc
}

func startNSServer(t *testing.T, chain registry.NetworkServiceRegistryServer) (u *url.URL, closeFunc func()) {
	s := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(s, chain)
	grpcutils.RegisterHealthServices(s, chain)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	closeFunc = func() {
		_ = l.Close()
	}
	go func() {
		_ = s.Serve(l)
	}()
	u, err = url.Parse("tcp://" + l.Addr().String())
	if err != nil {
		closeFunc()
	}
	require.Nil(t, err)
	return u, closeFunc
}

func testingNSEServerChain(ctx context.Context, u *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return next.NewNetworkServiceEndpointRegistryServer(
		proxy.NewNetworkServiceEndpointRegistryServer(u),
		connect.NewNetworkServiceEndpointRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient {
			return registry.NewNetworkServiceEndpointRegistryClient(cc)
		}, connect.WithExpirationDuration(time.Millisecond*100), connect.WithClientDialOptions(grpc.WithInsecure())),
	)
}

func testingNSServerChain(ctx context.Context, u *url.URL) registry.NetworkServiceRegistryServer {
	return next.NewNetworkServiceRegistryServer(
		proxy.NewNetworkServiceRegistryServer(u),
		connect.NewNetworkServiceRegistryServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
			return registry.NewNetworkServiceRegistryClient(cc)
		}, connect.WithExpirationDuration(time.Millisecond*100), connect.WithClientDialOptions(grpc.WithInsecure())),
	)
}
