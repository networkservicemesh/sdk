// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco Systems, Inc.
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
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/registry/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/registry/common/connect"
	"github.com/networkservicemesh/sdk/pkg/registry/common/dial"
	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/common/null"
	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/registry/core/chain"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
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

	cc, err := grpc.DialContext(ctx, grpcutils.URLToTarget(target), grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		if response.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING {
			return nil
		}
	}
	return ctx.Err()
}

func startTestNSEServers(ctx context.Context, t *testing.T) (url1, url2 *url.URL, cancel1, cancel2 context.CancelFunc) {
	var ctx1, ctx2 context.Context

	ctx1, cancel1 = context.WithCancel(ctx)

	url1 = &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSEServer(ctx1, url1, memory.NewNetworkServiceEndpointRegistryServer()))

	ctx2, cancel2 = context.WithCancel(ctx)

	url2 = &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	require.NoError(t, startNSEServer(ctx2, url2, memory.NewNetworkServiceEndpointRegistryServer()))

	require.NoError(t, waitNSEServerStarted(url1))
	require.NoError(t, waitNSEServerStarted(url2))

	return url1, url2, cancel1, cancel2
}

func TestConnectNSEServer_AllUnregister(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSEServers(ctx, t)
	defer cancel1()
	defer cancel2()

	ignoreCurrent := goleak.IgnoreCurrent()

	s := connect.NewNetworkServiceEndpointRegistryServer(
		chain.NewNetworkServiceEndpointRegistryClient(
			begin.NewNetworkServiceEndpointRegistryClient(),
			clientconn.NewNetworkServiceEndpointRegistryClient(),
			dial.NewNetworkServiceEndpointRegistryClient(ctx,
				dial.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
				dial.WithDialTimeout(time.Second),
			),
			connect.NewNetworkServiceEndpointRegistryClient(),
		),
	)

	_, err := s.Register(clienturlctx.WithClientURL(context.Background(), url1), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	_, err = s.Register(clienturlctx.WithClientURL(context.Background(), url2), &registry.NetworkServiceEndpoint{Name: "nse-1-1"})
	require.NoError(t, err)

	ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
	findSrv := streamchannel.NewNetworkServiceEndpointFindServer(clienturlctx.WithClientURL(ctx, url1), ch)
	err = s.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: "nse-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).GetNetworkServiceEndpoint().GetName(), "nse-1")

	findSrv = streamchannel.NewNetworkServiceEndpointFindServer(clienturlctx.WithClientURL(ctx, url2), ch)
	err = s.Find(&registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
		Name: "nse-1-1",
	}}, findSrv)
	require.NoError(t, err)
	require.Equal(t, (<-ch).GetNetworkServiceEndpoint().GetName(), "nse-1-1")

	_, err = s.Unregister(clienturlctx.WithClientURL(ctx, url1), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	_, err = s.Unregister(clienturlctx.WithClientURL(ctx, url2), &registry.NetworkServiceEndpoint{Name: "nse-1-1"})
	require.NoError(t, err)

	goleak.VerifyNone(t, ignoreCurrent)
}

func TestConnectNSEServer_AllDead_Register(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSEServers(ctx, t)

	s := connect.NewNetworkServiceEndpointRegistryServer(
		chain.NewNetworkServiceEndpointRegistryClient(
			begin.NewNetworkServiceEndpointRegistryClient(),
			clientconn.NewNetworkServiceEndpointRegistryClient(),
			dial.NewNetworkServiceEndpointRegistryClient(ctx,
				dial.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
				dial.WithDialTimeout(time.Second),
			),
			connect.NewNetworkServiceEndpointRegistryClient(),
		),
	)
	_, err := s.Register(clienturlctx.WithClientURL(ctx, url1), &registry.NetworkServiceEndpoint{Name: "nse-1"})
	require.NoError(t, err)

	_, err = s.Register(clienturlctx.WithClientURL(ctx, url2), &registry.NetworkServiceEndpoint{Name: "nse-1-1"})
	require.NoError(t, err)

	cancel1()
	cancel2()

	var i int
	for err, i = goleak.Find(), 0; err != nil && i < 3; err, i = goleak.Find(), i+1 {
	}
}

func TestConnectNSEServer_AllDead_WatchingFind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url1, url2, cancel1, cancel2 := startTestNSEServers(ctx, t)

	s := connect.NewNetworkServiceEndpointRegistryServer(
		chain.NewNetworkServiceEndpointRegistryClient(
			begin.NewNetworkServiceEndpointRegistryClient(),
			clientconn.NewNetworkServiceEndpointRegistryClient(),
			dial.NewNetworkServiceEndpointRegistryClient(ctx,
				dial.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
				dial.WithDialTimeout(time.Second),
			),
			connect.NewNetworkServiceEndpointRegistryClient(),
		),
	)

	errCh := make(chan error, 2)

	go func() {
		ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
		findSrv := streamchannel.NewNetworkServiceEndpointFindServer(clienturlctx.WithClientURL(ctx, url1), ch)
		err := s.Find(&registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
			Watch:                  true,
		}, findSrv)
		errCh <- err
	}()

	go func() {
		ch := make(chan *registry.NetworkServiceEndpointResponse, 1)
		findSrv := streamchannel.NewNetworkServiceEndpointFindServer(clienturlctx.WithClientURL(ctx, url2), ch)
		err := s.Find(&registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
			Watch:                  true,
		}, findSrv)
		errCh <- err
	}()

	cancel1()
	cancel2()

	<-errCh
	<-errCh

	for err, i := goleak.Find(), 0; err != nil && i < 3; err, i = goleak.Find(), i+1 {
	}
}

func Test_ConenctNSEChain_Find(t *testing.T) {
	for depth := 2; depth < 11; depth++ {
		for killIndex := 1; killIndex < depth; killIndex++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			urls := make([]*url.URL, depth)

			servers := make([]*struct {
				registry.NetworkServiceEndpointRegistryServer
				kill func()
			}, depth)

			for i := 0; i < depth; i++ {
				serverCtx, serverCancel := context.WithCancel(ctx)

				servers[i] = &struct {
					registry.NetworkServiceEndpointRegistryServer
					kill func()
				}{
					kill: serverCancel,
				}

				urls[i] = new(url.URL)

				require.NoError(t,
					startNSEServer(
						serverCtx,
						urls[i],
						servers[i],
					),
				)
			}

			for i := 0; i < depth-1; i++ {
				servers[i].NetworkServiceEndpointRegistryServer = chain.NewNetworkServiceEndpointRegistryServer(
					clienturl.NewNetworkServiceEndpointRegistryServer(urls[i+1]),
					connect.NewNetworkServiceEndpointRegistryServer(
						chain.NewNetworkServiceEndpointRegistryClient(
							begin.NewNetworkServiceEndpointRegistryClient(),
							clientconn.NewNetworkServiceEndpointRegistryClient(),
							dial.NewNetworkServiceEndpointRegistryClient(ctx,
								dial.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())),
								dial.WithDialTimeout(time.Second),
							),
							connect.NewNetworkServiceEndpointRegistryClient(),
						),
					),
				)
			}

			servers[len(servers)-1].NetworkServiceEndpointRegistryServer = memory.NewNetworkServiceEndpointRegistryServer()

			c := adapters.NetworkServiceEndpointServerToClient(servers[0].NetworkServiceEndpointRegistryServer)

			_, err := c.Register(ctx, &registry.NetworkServiceEndpoint{
				Name: "testing",
			})

			require.NoError(t, err)

			stream, err := c.Find(ctx, &registry.NetworkServiceEndpointQuery{
				Watch: true,
				NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
					Name: "testing",
				},
			})
			require.NoError(t, err)

			_, err = stream.Recv()
			require.NoError(t, err)

			servers[killIndex].kill()

			_, err = stream.Recv()
			require.Error(t, err)
		}
	}
}
