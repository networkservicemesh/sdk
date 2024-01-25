// Copyright (c) 2020-2024 Doc.ai and/or its affiliates.
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

package timeout_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/refresh"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injectclock"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/inject/injecterror"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/clockmock"
)

const (
	clientName   = "client"
	serverName   = "server"
	tokenTimeout = 15 * time.Minute
	testWait     = 150 * time.Millisecond
	testTick     = testWait / 100
)

func testClient(
	ctx context.Context,
	client networkservice.NetworkServiceClient,
	server networkservice.NetworkServiceServer,
	duration time.Duration,
	clk clock.Clock,
) networkservice.NetworkServiceClient {
	return next.NewNetworkServiceClient(
		updatepath.NewClient(clientName),
		begin.NewClient(),
		metadata.NewClient(),
		injectclock.NewClient(clk),
		client,
		adapters.NewServerToClient(
			next.NewNetworkServiceServer(
				updatetoken.NewServer(func(_ credentials.AuthInfo) (string, time.Time, error) {
					return "token", clock.FromContext(ctx).Now().Add(duration), nil
				}),
				begin.NewServer(),
				metadata.NewServer(),
				new(remoteServer), // <-- GRPC invocation
				updatepath.NewServer(serverName),
				timeout.NewServer(ctx),
				server,
			),
		),
	)
}

func TestTimeoutServer_Request(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	connServer := newConnectionsServer(t)

	client := testClient(ctx,
		kernel.NewClient(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: connServer,
		}),
		tokenTimeout,
		clockMock,
	)

	_, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)

	require.Eventually(t, connServer.validator(1, 0), testWait, testTick)

	clockMock.Add(tokenTimeout / 2)
	require.Never(t, connServer.validator(0, 1), testWait, testTick)

	clockMock.Add(tokenTimeout / 2)
	require.Eventually(t, connServer.validator(0, 1), testWait, testTick)
}

func TestTimeoutServer_CloseBeforeTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	connServer := newConnectionsServer(t)

	client := testClient(ctx,
		kernel.NewClient(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: connServer,
		}),
		tokenTimeout,
		clockMock,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Condition(t, connServer.validator(0, 1))

	// ensure there will be no double Close
	clockMock.Add(tokenTimeout)
	require.Never(t, func() bool {
		return !connServer.validator(0, 1)()
	}, testWait, testTick)
}

func TestTimeoutServer_CloseAfterTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	connServer := newConnectionsServer(t)

	client := testClient(ctx,
		kernel.NewClient(),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: connServer,
		}),
		tokenTimeout,
		clockMock,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	clockMock.Add(tokenTimeout)
	require.Eventually(t, connServer.validator(0, 1), testWait, testTick)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Condition(t, connServer.validator(0, 1))
}

func raceTestRequest() *networkservice.NetworkServiceRequest {
	return &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Path: &networkservice.Path{
				PathSegments: []*networkservice.PathSegment{
					{
						Name: clientName,
						Id:   "client-id",
					},
					{
						Name: serverName,
						Id:   "server-id",
					},
				},
			},
		},
	}
}

func TestTimeoutServer_RaceTest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connServer := newConnectionsServer(t)

	client := testClient(ctx, null.NewClient(), connServer, 0, clock.FromContext(ctx))

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := client.Request(ctx, raceTestRequest())
			assert.NoError(t, err)

			_, err = client.Close(ctx, conn)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestTimeoutServer_RefreshFailure(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	connServer := newConnectionsServer(t)

	client := testClient(
		ctx,
		next.NewNetworkServiceClient(
			begin.NewClient(),
			metadata.NewClient(),
			refresh.NewClient(ctx),
		),
		next.NewNetworkServiceServer(
			injecterror.NewServer(
				injecterror.WithRequestErrorTimes(1, -1),
				injecterror.WithCloseErrorTimes(),
			),
			updatetoken.NewServer(func(_ credentials.AuthInfo) (string, time.Time, error) {
				return "test", clock.FromContext(ctx).Now().Add(tokenTimeout), nil
			}),
			connServer,
		),
		tokenTimeout,
		clockMock,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	clockMock.Add(tokenTimeout)
	require.Eventually(t, connServer.validator(0, 1), testWait, testTick)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Condition(t, connServer.validator(0, 1))
}

func TestTimeoutServer_CloseFailure(t *testing.T) {
	t.Skip("consider to remote this one")
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clockMock := clockmock.New(ctx)
	ctx = clock.WithClock(ctx, clockMock)

	connServer := newConnectionsServer(t)

	client := testClient(
		ctx,
		null.NewClient(),
		next.NewNetworkServiceServer(
			injecterror.NewServer(
				injecterror.WithError(errors.WithStack(context.DeadlineExceeded)),
				injecterror.WithRequestErrorTimes(),
				injecterror.WithCloseErrorTimes(0)),

			connServer,
		),
		tokenTimeout,
		clockMock,
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	_, err = client.Close(ctx, conn)
	require.Error(t, err)
	require.Condition(t, connServer.validator(1, 0))

	clockMock.Add(tokenTimeout)
	require.Eventually(t, connServer.validator(0, 1), testWait, testTick)
}

type remoteServer struct{}

func (s *remoteServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	return next.Server(ctx).Request(ctx, request.Clone())
}

func (s *remoteServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	return next.Server(ctx).Close(ctx, conn.Clone())
}

type connectionsServer struct {
	t           *testing.T
	lock        sync.Mutex
	connections map[string]bool
}

func newConnectionsServer(t *testing.T) *connectionsServer {
	return &connectionsServer{
		t:           t,
		connections: map[string]bool{},
	}
}

func (s *connectionsServer) validator(open, closed int) func() bool {
	return func() bool {
		s.lock.Lock()
		defer s.lock.Unlock()

		var connsOpen, connsClosed int
		for _, isOpen := range s.connections {
			if isOpen {
				connsOpen++
			} else {
				connsClosed++
			}
		}

		if connsOpen != open || connsClosed != closed {
			return false
		}
		return true
	}
}

func (s *connectionsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.lock.Lock()

	s.connections[request.GetConnection().GetId()] = true

	s.lock.Unlock()

	return next.Server(ctx).Request(ctx, request)
}

func (s *connectionsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.lock.Lock()

	s.connections[conn.GetId()] = false

	s.lock.Unlock()

	return next.Server(ctx).Close(ctx, conn)
}
