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

	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
)

func startNSServer(ctx context.Context, listenOn *url.URL) error {
	grpcServer := grpc.NewServer()

	server := memory.NewNetworkServiceRegistryServer()
	registry.RegisterNetworkServiceRegistryServer(grpcServer, server)
	grpcutils.RegisterHealthServices(grpcServer, server)

	errCh := grpcutils.ListenAndServe(ctx, listenOn, grpcServer)
	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func waitNSServerStarted(target *url.URL) error {
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
		Service: registry.ServiceNames(null.NewNetworkServiceRegistryServer())[0],
	}

	client := grpc_health_v1.NewHealthClient(cc)
	for ctx.Err() == nil {
		respons, err := client.Check(ctx, healthCheckRequest)
		if err != nil {
			return err
		}
		if respons.Status == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		}
	}
	return ctx.Err()
}

func startTestNSServers(ctx context.Context, t *testing.T) (url1, url2 *url.URL, cancel1, cancel2 context.CancelFunc) {
	var ctx1, ctx2 context.Context

	ctx1, cancel1 = context.WithCancel(ctx)

	url1 = &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSServer(ctx1, url1))

	ctx2, cancel2 = context.WithCancel(ctx)

	url2 = &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSServer(ctx2, url2))

	require.NoError(t, waitNSServerStarted(url1))
	require.NoError(t, waitNSServerStarted(url2))

	return url1, url2, cancel1, cancel2
}

func TestConnectNSServer_AllUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSServers(ctx, t)
	defer cancel1()
	defer cancel2()

	ignoreCurrent := goleak.IgnoreCurrent()

	s := connect.NewNetworkServiceRegistryServer(ctx, func(_ context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
		return registry.NewNetworkServiceRegistryClient(cc)
	}, grpc.WithInsecure())

	_, err := s.Register(clienturlctx.WithClientURL(context.Background(), url1), &registry.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	_, err = s.Register(clienturlctx.WithClientURL(context.Background(), url2), &registry.NetworkService{Name: "ns-1-1"})
	require.NoError(t, err)

	ch := make(chan *registry.NetworkService, 1)
	findSrv := streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(ctx, url1), ch)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{
		Name: "ns-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).Name, "ns-1")

	findSrv = streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(ctx, url2), ch)
	err = s.Find(&registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{
		Name: "ns-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).Name, "ns-1-1")

	_, err = s.Unregister(clienturlctx.WithClientURL(ctx, url1), &registry.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	_, err = s.Unregister(clienturlctx.WithClientURL(ctx, url2), &registry.NetworkService{Name: "ns-1-1"})
	require.NoError(t, err)

	goleak.VerifyNone(t, ignoreCurrent)
}

func TestConnectNSServer_AllDead_Register(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSServers(ctx, t)

	s := connect.NewNetworkServiceRegistryServer(ctx, func(_ context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
		return registry.NewNetworkServiceRegistryClient(cc)
	}, grpc.WithInsecure())

	_, err := s.Register(clienturlctx.WithClientURL(ctx, url1), &registry.NetworkService{Name: "ns-1"})
	require.NoError(t, err)

	_, err = s.Register(clienturlctx.WithClientURL(ctx, url2), &registry.NetworkService{Name: "ns-1-1"})
	require.NoError(t, err)

	cancel1()
	cancel2()

	var i int
	for err, i = goleak.Find(), 0; err != nil && i < 3; err, i = goleak.Find(), i+1 {
	}
}

func TestConnectNSServer_AllDead_WatchingFind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSServers(ctx, t)

	s := connect.NewNetworkServiceRegistryServer(ctx, func(_ context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceRegistryClient {
		return registry.NewNetworkServiceRegistryClient(cc)
	}, grpc.WithInsecure())

	go func() {
		ch := make(chan *registry.NetworkService, 1)
		findSrv := streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(ctx, url1), ch)
		err := s.Find(&registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
			Watch:          true,
		}, findSrv)
		require.Error(t, err)
	}()

	go func() {
		ch := make(chan *registry.NetworkService, 1)
		findSrv := streamchannel.NewNetworkServiceFindServer(clienturlctx.WithClientURL(ctx, url2), ch)
		err := s.Find(&registry.NetworkServiceQuery{
			NetworkService: new(registry.NetworkService),
			Watch:          true,
		}, findSrv)
		require.Error(t, err)
	}()

	cancel1()
	cancel2()

	for err, i := goleak.Find(), 0; err != nil && i < 3; err, i = goleak.Find(), i+1 {
	}
}
