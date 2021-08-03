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

package checkconn_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/checkconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/checks/checkcontext"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestCheckConn(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	eventCh := make(chan *networkservice.ConnectionEvent, 1)
	defer close(eventCh)

	monitorServer := eventchannel.NewMonitorServer(eventCh)

	counter := newCounterServer()
	server := chain.NewNetworkServiceServer(
		checkcontext.NewServer(t, func(t *testing.T, ctx context.Context) {
			require.Nil(t, ctx.Err())
		}),
		updatetoken.NewServer(sandbox.GenerateTestToken),
		updatepath.NewServer("testServer"),
		monitor.NewServer(ctx, &monitorServer),
		updatetoken.NewServer(sandbox.GenerateTestToken),
		counter,
	)

	// Create chain element that aborts returning back connection
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errServer := newErrorServer(cancel)
	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("testClient"),
		checkconn.NewClient(adapters.NewMonitorServerToClient(monitorServer)),
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				errServer,
				server,
			),
		),
	)

	// Server increments the counter but returns false on request
	// checkconn chain element should call close and release resources
	_, err := client.Request(cancelCtx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "ns-1",
		},
	})
	require.Error(t, err)
	require.Equal(t, counter.requests, counter.closes)

	// Allow requests
	errServer.disable()
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "ns-1",
		},
	})
	require.NoError(t, err)
	require.NotEqual(t, counter.requests, counter.closes)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Equal(t, counter.requests, counter.closes)
}

// errorServer - simulates error on the server side
type errorServer struct {
	enabled   bool
	ctxCancel context.CancelFunc
}

func newErrorServer(ctxCancel context.CancelFunc) *errorServer {
	return &errorServer{
		enabled:   true,
		ctxCancel: ctxCancel,
	}
}

func (s *errorServer) disable() {
	s.enabled = false
}

func (s *errorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, e := next.Server(ctx).Request(ctx, request.Clone())
	if s.enabled {
		s.ctxCancel()
		return nil, errors.New("request error on the server")
	}
	return conn, e
}

func (s *errorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn.Clone())
}

func newCounterServer() *counterServer {
	return &counterServer{}
}

type counterServer struct {
	requests int32
	closes   int32
}

func (s *counterServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	atomic.AddInt32(&s.requests, 1)
	return next.Server(ctx).Request(ctx, request)
}

func (s *counterServer) Close(ctx context.Context, conn *networkservice.Connection) (*emptypb.Empty, error) {
	atomic.AddInt32(&s.closes, 1)
	return next.Server(ctx).Close(ctx, conn)
}
