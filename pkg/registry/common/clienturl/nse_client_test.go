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

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/memory"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TestClientURL_NewNetworkServiceEndpointRegistryClient(t *testing.T) {
	serverChain := memory.NewNetworkServiceEndpointRegistryServer()
	_, err := serverChain.Register(context.Background(), &registry.NetworkServiceEndpoint{Name: "ns-1"})
	require.Nil(t, err)
	s := grpc.NewServer()
	registry.RegisterNetworkServiceEndpointRegistryServer(s, serverChain)
	grpcutils.RegisterHealthServices(s, serverChain)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	defer func() {
		_ = l.Close()
	}()
	go func() {
		_ = s.Serve(l)
	}()
	defer s.Stop()
	u, err := url.Parse("tcp://" + l.Addr().String())
	require.Nil(t, err)
	ctx := clienturlctx.WithClientURL(context.Background(), u)
	client := clienturl.NewNetworkServiceEndpointRegistryClient(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient {
		return registry.NewNetworkServiceEndpointRegistryClient(cc)
	}, grpc.WithInsecure())

	stream, err := client.Find(context.Background(), &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: "ns-1"}}, grpc.WaitForReady(true))
	require.Nil(t, err)
	list := registry.ReadNetworkServiceEndpointList(stream)
	require.NotEmpty(t, list)
}
