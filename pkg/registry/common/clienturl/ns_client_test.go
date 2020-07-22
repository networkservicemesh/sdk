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

package clienturl_test

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TestClientURL_NewNetworkServiceRegistryClient(t *testing.T) {
	serverChain := memory.NewNetworkServiceRegistryServer()
	_, err := serverChain.Register(context.Background(), &registry.NetworkService{Name: "ns-1"})
	require.Nil(t, err)
	s := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(s, serverChain)
	grpcutils.RegisterHealthServices(s, serverChain)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() {
		_ = l.Close()
	}()
	go func() {
		_ = s.Serve(l)
	}()
	u, err := url.Parse("tcp://" + l.Addr().String())
	require.Nil(t, err)

	ctx, clientCancel := context.WithCancel(context.Background())
	defer clientCancel()
	ctx = clienturl.WithClientURL(ctx, u)

	client := clienturl.NewNetworkServiceRegistryClient(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
		return registry.NewNetworkServiceRegistryClient(cc)
	}, grpc.WithInsecure())
	stream, err := client.Find(context.Background(), &registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: "ns-1"}}, grpc.WaitForReady(true))
	require.Nil(t, err)
	list := registry.ReadNetworkServiceList(stream)
	require.NotEmpty(t, list)
}
