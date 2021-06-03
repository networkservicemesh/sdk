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

package connectto_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connectto"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startNSEServer(ctx context.Context, listenOn *url.URL, server registry.NetworkServiceEndpointRegistryServer) error {
	grpcServer := grpc.NewServer()

	registry.RegisterNetworkServiceEndpointRegistryServer(grpcServer, server)
	grpcutils.RegisterHealthServices(grpcServer, server)

	errCh := grpcutils.ListenAndServe(ctx, listenOn, grpcServer)
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func waitNSEServerStarted(target *url.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(target), grpc.WithBlock(), grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer func() {
		_ = cc.Close()
	}()

	healthCheckRequest := &grpc_health_v1.HealthCheckRequest{
		Service: registry.ServiceNames(null.NewNetworkServiceEndpointRegistryServer())[0],
	}

	client := grpc_health_v1.NewHealthClient(cc)
	for ctx.Err() == nil {
		response, err := client.Check(ctx, healthCheckRequest)
		if err != nil {
			return err
		}
		if response.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		}
	}
	return ctx.Err()
}

func TestConnectNSEClient(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSEServer(ctx, u, mem))
	require.NoError(t, waitNSEServerStarted(u))

	// 1. Register remote NSE
	_, err := mem.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-remote"})
	require.NoError(t, err)

	c := connectto.NewNetworkServiceEndpointRegistryClient(ctx, grpcutils.URLToTarget(u),
		connectto.WithDialOptions(grpc.WithInsecure()),
	)

	// 2. Register local NSE
	_, err = c.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-local"})
	require.NoError(t, err)

	// 3. Find both local, remote NSEs from client
	stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	})
	require.NoError(t, err)

	var nseNames []string
	for _, nse := range registry.ReadNetworkServiceEndpointList(stream) {
		nseNames = append(nseNames, nse.Name)
	}
	require.Len(t, nseNames, 2)
	require.Subset(t, []string{"nse-remote", "nse-local"}, nseNames)

	// 4. Unregister remote NSE from client
	_, err = c.Unregister(ctx, &registry.NetworkServiceEndpoint{Name: "nse-remote"})
	require.NoError(t, err)

	// 5. Find only local NSE in memory
	ch := make(chan *registry.NetworkServiceEndpoint, 2)
	err = mem.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	require.NoError(t, err)

	require.Len(t, ch, 1)
	require.Equal(t, "nse-local", (<-ch).Name)
}

func TestConnectNSEClient_Restart(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCtx, serverCancel := context.WithCancel(ctx)

	mem := memory.NewNetworkServiceEndpointRegistryServer()

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSEServer(serverCtx, u, mem))
	require.NoError(t, waitNSEServerStarted(u))

	c := connectto.NewNetworkServiceEndpointRegistryClient(ctx, grpcutils.URLToTarget(u),
		connectto.WithDialOptions(grpc.WithInsecure()),
	)

	// 1. Register NSE-1 with client
	_, err := c.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	// 2. Restart remote
	serverCancel()
	require.Eventually(t, grpcutils.CheckURLFree(grpcutils.URLToTarget(u)), time.Second, 10*time.Millisecond)

	require.NoError(t, startNSEServer(ctx, u, mem))
	require.NoError(t, waitNSEServerStarted(u))

	// 3. Register NSE-2 with client
	require.Eventually(t, func() bool {
		_, err = c.Register(ctx, &registry.NetworkServiceEndpoint{Name: "nse-2"})
		return err == nil
	}, time.Second, 10*time.Millisecond)

	// 4. Find both NSE-1, NSE-2 in memory
	ch := make(chan *registry.NetworkServiceEndpoint, 2)
	err = mem.Find(&registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
	}, streamchannel.NewNetworkServiceEndpointFindServer(ctx, ch))
	require.NoError(t, err)

	var nseNames []string
	for i := len(ch); i > 0; i-- {
		nseNames = append(nseNames, (<-ch).Name)
	}
	require.Len(t, nseNames, 2)
	require.Subset(t, []string{"nse-1", "nse-2"}, nseNames)
}
