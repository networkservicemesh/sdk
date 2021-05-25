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

package connect_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startNSServer(t *testing.T) (*url.URL, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	serverChain := memory.NewNetworkServiceRegistryServer()
	s := grpc.NewServer()
	registry.RegisterNetworkServiceRegistryServer(s, serverChain)
	grpcutils.RegisterHealthServices(s, serverChain)

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	select {
	case err := <-grpcutils.ListenAndServe(ctx, u, s):
		require.NoError(t, err)
	default:
	}

	return u, cancel
}

func TestConnect_NewNetworkServiceRegistryServer(t *testing.T) {
	url1, closeServer1 := startNSServer(t)
	url2, closeServer2 := startNSServer(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := connect.NewNetworkServiceRegistryServer(ctx, func(_ context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
		return registry.NewNetworkServiceRegistryClient(cc)
	}, grpc.WithInsecure())

	_, err := s.Register(clienturlctx.WithClientURL(context.Background(), url1), &registry.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	_, err = s.Register(clienturlctx.WithClientURL(context.Background(), url2), &registry.NetworkService{Name: "ns-1-1"})
	require.NoError(t, err)

	ch := make(chan *registry.NetworkService, 1)
	findSrv := streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(context.Background(), url1), ch)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{
		Name: "ns-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).Name, "ns-1")

	findSrv = streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(context.Background(), url2), ch)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{
		Name: "ns-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).Name, "ns-1-1")

	closeServer1()
	closeServer2()

	require.Eventually(t, func() bool {
		return goleak.Find(goleak.IgnoreTopFunction("github.com/stretchr/testify/assert.Eventually")) == nil
	}, time.Second, 100*time.Millisecond)
}
