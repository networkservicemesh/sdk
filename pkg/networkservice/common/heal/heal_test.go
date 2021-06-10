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

package heal_test

import (
	"context"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func startRemoteServer(ctx context.Context, t *testing.T, expireDuration time.Duration) (*url.URL, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	server := endpoint.NewServer(ctx, sandbox.GenerateExpiringToken(expireDuration), endpoint.WithName("remote"))

	grpcServer := grpc.NewServer()
	server.Register(grpcServer)

	u := &url.URL{Scheme: "tcp", Host: "127.0.0.1:0"}
	select {
	case err := <-grpcutils.ListenAndServe(ctx, u, grpcServer):
		require.NoError(t, err)
	default:
	}

	return u, cancel
}

func TestHeal_CloseChain(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	remoteURL, remoteCancel := startRemoteServer(ctx, t, 0)
	defer remoteCancel()

	counter := new(counterServer)

	serverChain := new(networkservice.NetworkServiceClient)
	*serverChain = adapters.NewServerToClient(
		next.NewNetworkServiceServer(
			updatepath.NewServer("server"),
			updatetoken.NewServer(sandbox.GenerateTestToken),
			heal.NewServer(ctx,
				heal.WithOnHeal(serverChain)),
			clienturl.NewServer(remoteURL),
			connect.NewServer(ctx, func(ctx context.Context, cc grpc.ClientConnInterface) networkservice.NetworkServiceClient {
				return next.NewNetworkServiceClient(
					heal.NewClient(ctx, networkservice.NewMonitorConnectionClient(cc)),
					networkservice.NewNetworkServiceClient(cc),
				)
			}, connect.WithDialOptions(grpc.WithInsecure())),
			counter,
		),
	)

	client := next.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		*serverChain,
	)

	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: new(networkservice.Connection),
	})
	require.NoError(t, err)

	remoteCancel()

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&counter.closes) == 1
	}, time.Second, 10*time.Millisecond)
}

type counterServer struct {
	closes int32
}

func (s *counterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return next.Server(ctx).Request(ctx, request)
}

func (s *counterServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	atomic.AddInt32(&s.closes, 1)

	return next.Server(ctx).Close(ctx, conn)
}
