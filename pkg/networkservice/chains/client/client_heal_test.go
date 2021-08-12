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

package client_test

import (
	"context"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/count"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func startServer(ctx context.Context, t *testing.T, serverURL *url.URL, opts ...endpoint.Option) networkservice.NetworkServiceServer {
	nse := endpoint.NewServer(ctx, sandbox.GenerateTestToken, opts...)

	select {
	case err := <-endpoint.Serve(ctx, serverURL, nse):
		require.NoError(t, err)
	default:
	}

	return nse
}

func TestClient_Heal(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	startServer(serverCtx, t, serverURL)

	nsc := client.NewClient(ctx,
		serverURL,
		client.WithDialOptions(sandbox.DialOptions()...),
		client.WithDialTimeout(time.Second),
	)

	_, err := nsc.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)

	serverCancel()
	require.Eventually(t, func() bool {
		return sandbox.CheckURLFree(serverURL)
	}, time.Second, time.Millisecond*10)
	require.NoError(t, ctx.Err())

	startServer(ctx, t, serverURL)

	require.Eventually(t, func() bool {
		_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{})
		return err == nil
	}, time.Second*2, time.Millisecond*50)
}

func TestClient_StopHealingOnFailure(t *testing.T) {
	var samples = []struct {
		name         string
		optsSupplier func(counter networkservice.NetworkServiceClient) []client.Option
	}{
		{
			name: "Authorize failure",
			optsSupplier: func(counter networkservice.NetworkServiceClient) []client.Option {
				return []client.Option{
					client.WithAuthorizeClient(new(refreshFailureClient)),
					client.WithAdditionalFunctionality(counter),
				}
			},
		},
		{
			name: "Additional functionality failure",
			optsSupplier: func(counter networkservice.NetworkServiceClient) []client.Option {
				return []client.Option{
					client.WithAdditionalFunctionality(
						new(refreshFailureClient),
						counter,
					),
				}
			},
		},
	}

	for _, sample := range samples {
		// nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			testStopHealingOnFailure(t, func(ctx context.Context, serverURL *url.URL, counter networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
				return client.NewClient(ctx,
					serverURL,
					append([]client.Option{
						client.WithDialOptions(sandbox.DialOptions()...),
						client.WithDialTimeout(time.Second),
					}, sample.optsSupplier(counter)...)...,
				)
			})
		})
	}
}

func TestClientFactory_StopHealingOnFailure(t *testing.T) {
	var samples = []struct {
		name string
		opts []client.Option
	}{
		{
			name: "Authorize failure",
			opts: []client.Option{
				client.WithAuthorizeClient(new(refreshFailureClient)),
			},
		},
		{
			name: "Additional functionality failure",
			opts: []client.Option{
				client.WithAdditionalFunctionality(new(refreshFailureClient)),
			},
		},
	}

	for _, sample := range samples {
		// nolint:scopelint
		t.Run(sample.name, func(t *testing.T) {
			testStopHealingOnFailure(t, func(ctx context.Context, serverURL *url.URL, counter networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
				clientServerURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}

				clientServer := new(struct {
					networkservice.NetworkServiceServer
				})
				clientServer.NetworkServiceServer = startServer(ctx, t, clientServerURL,
					endpoint.WithName("name"),
					endpoint.WithAdditionalFunctionality(
						heal.NewServer(ctx,
							heal.WithOnHeal(addressof.NetworkServiceClient(adapters.NewServerToClient(clientServer)))),
						adapters.NewClientToServer(counter),
						clienturl.NewServer(serverURL),
						connect.NewServer(ctx, client.NewClientFactory(
							append([]client.Option{
								client.WithName("name"),
							}, sample.opts...)...),
							connect.WithDialOptions(sandbox.DialOptions()...),
							connect.WithDialTimeout(time.Second),
						),
					),
				)

				return client.NewClient(ctx,
					clientServerURL,
					client.WithDialOptions(sandbox.DialOptions()...),
					client.WithDialTimeout(time.Second),
				)
			})
		})
	}
}

func testStopHealingOnFailure(
	t *testing.T,
	clientSupplier func(ctx context.Context, serverURL *url.URL, counter networkservice.NetworkServiceClient) networkservice.NetworkServiceClient,
) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	serverURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	startServer(ctx, t, serverURL)

	counter := new(count.Client)
	nsc := clientSupplier(ctx, serverURL, counter)

	conn, err := nsc.Request(ctx, new(networkservice.NetworkServiceRequest))
	require.NoError(t, err)

	_, err = nsc.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: conn.Clone(),
	})
	require.Errorf(t, err, "refresh error")

	require.Never(t, func() bool {
		// 1.  Request
		// 2.  Failed refresh Request
		// 3+. Heal Requests
		return counter.Requests() > 2
	}, time.Second*2, time.Millisecond*50)
}

type refreshFailureClient struct {
	flag int32
}

func (c *refreshFailureClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if err = ctx.Err(); err == nil && atomic.CompareAndSwapInt32(&c.flag, 0, 1) {
		return conn, nil
	}

	_, _ = next.Client(ctx).Close(ctx, conn, opts...)

	if err == nil {
		err = errors.New("refresh error")
	}
	return nil, err
}

func (c *refreshFailureClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return next.Client(ctx).Close(ctx, conn, opts...)
}
