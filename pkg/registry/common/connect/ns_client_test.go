// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func TestConnectNSClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceRegistryServer()

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSServer(ctx, u, mem))
	require.NoError(t, waitNSServerStarted(u))

	// 1. Register remote NS
	_, err := mem.Register(ctx, &registry.NetworkService{Name: "ns-remote"})
	require.NoError(t, err)

	c := connect.NewNetworkServiceRegistryClient(ctx, grpcutils.URLToTarget(u),
		connect.WithDialOptions(grpc.WithInsecure()),
	)

	// 2. Register local NS
	_, err = c.Register(ctx, &registry.NetworkService{Name: "ns-local"})
	require.NoError(t, err)

	// 3. Find both local, remote NSs from client
	stream, err := c.Find(ctx, &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
	})
	require.NoError(t, err)

	var nsNames []string
	for _, ns := range registry.ReadNetworkServiceList(stream) {
		nsNames = append(nsNames, ns.Name)
	}
	require.Len(t, nsNames, 2)
	require.Subset(t, []string{"ns-remote", "ns-local"}, nsNames)

	// 4. Unregister remote NS from client
	_, err = c.Unregister(ctx, &registry.NetworkService{Name: "ns-remote"})
	require.NoError(t, err)

	// 5. Find only local NS in memory
	ch := make(chan *registry.NetworkService, 2)
	err = mem.Find(&registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	require.NoError(t, err)

	require.Len(t, ch, 1)
	require.Equal(t, "ns-local", (<-ch).Name)
}

func TestConnectNSClient_Restart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCtx, serverCancel := context.WithCancel(ctx)

	mem := memory.NewNetworkServiceRegistryServer()

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSServer(serverCtx, u, mem))
	require.NoError(t, waitNSServerStarted(u))

	c := connect.NewNetworkServiceRegistryClient(ctx, grpcutils.URLToTarget(u),
		connect.WithDialOptions(grpc.WithInsecure()),
	)

	// 1. Register NS-1 with client
	_, err := c.Register(ctx, &registry.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	// 2. Restart remote
	serverCancel()
	require.Eventually(t, grpcutils.CheckURLFree(grpcutils.URLToTarget(u)), time.Second, 10*time.Millisecond)

	require.NoError(t, startNSServer(ctx, u, mem))
	require.NoError(t, waitNSServerStarted(u))

	// 3. Register NS-2 with client
	require.Eventually(t, func() bool {
		_, err = c.Register(ctx, &registry.NetworkService{Name: "ns-2"})
		return err == nil
	}, time.Second, 10*time.Millisecond)

	// 4. Find both NS-1, NS-2 in memory
	ch := make(chan *registry.NetworkService, 2)
	err = mem.Find(&registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
	}, streamchannel.NewNetworkServiceFindServer(ctx, ch))
	require.NoError(t, err)

	var nsNames []string
	for i := len(ch); i > 0; i-- {
		nsNames = append(nsNames, (<-ch).Name)
	}
	require.Len(t, nsNames, 2)
	require.Subset(t, []string{"ns-1", "ns-2"}, nsNames)
}
