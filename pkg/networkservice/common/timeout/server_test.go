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

package timeout_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/networkservicemesh/sdk/pkg/tools/logger"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/serialize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	clientName    = "client"
	serverName    = "server"
	tokenTimeout  = 100 * time.Millisecond
	waitFor       = 10 * tokenTimeout
	tick          = 10 * time.Millisecond
	serverID      = "server-id"
	parallelCount = 1000
)

func testClient(ctx context.Context, server networkservice.NetworkServiceServer, duration time.Duration) networkservice.NetworkServiceClient {
	return chain.NewNetworkServiceClient(
		updatepath.NewClient(clientName),
		updatetoken.NewClient(func(_ credentials.AuthInfo) (string, time.Time, error) {
			return "token", time.Now().Add(duration), nil
		}),
		kernel.NewClient(),
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				updatepath.NewServer(serverName),
				serialize.NewServer(),
				timeout.NewServer(ctx),
				mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
					kernelmech.MECHANISM: server,
				}),
			),
		),
	)
}

func TestTimeoutServer_Request(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connServer := newConnectionsServer(t)

	_, err := testClient(ctx, connServer, tokenTimeout).Request(logger.WithLog(ctx), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	require.Eventually(t, connServer.validator(0, 1), waitFor, tick)
}

func TestTimeoutServer_Close_BeforeTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connServer := newConnectionsServer(t)

	client := testClient(ctx, connServer, tokenTimeout)

	ctx = logger.WithLog(ctx)
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Condition(t, connServer.validator(0, 1))

	// ensure there will be no double Close
	<-time.After(waitFor)
}

func TestTimeoutServer_Close_AfterTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connServer := newConnectionsServer(t)

	client := testClient(ctx, connServer, tokenTimeout)

	ctx = logger.WithLog(ctx)
	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.Condition(t, connServer.validator(1, 0))

	require.Eventually(t, connServer.validator(0, 1), waitFor, tick)

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
	require.Condition(t, connServer.validator(0, 1))
}

func stressTestRequest() *networkservice.NetworkServiceRequest {
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
						Id:   serverID,
					},
				},
			},
		},
	}
}

func TestTimeoutServer_StressTest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	connServer := newConnectionsServer(t)

	client := testClient(ctx, connServer, 0)

	ctx = logger.WithLog(ctx)
	wg := new(sync.WaitGroup)
	wg.Add(parallelCount)
	for i := 0; i < parallelCount; i++ {
		go func() {
			defer wg.Done()
			conn, err := client.Request(ctx, stressTestRequest())
			assert.NoError(t, err)
			_, err = client.Close(ctx, conn)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
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

	connID := request.GetConnection().GetId()

	s.connections[connID] = true

	s.lock.Unlock()

	return next.Server(ctx).Request(ctx, request)
}

func (s *connectionsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.lock.Lock()

	connID := conn.GetId()

	if !s.connections[connID] {
		assert.Fail(s.t, "closing not opened connection: %v", connID)
	} else {
		s.connections[connID] = false
	}

	s.lock.Unlock()

	return next.Server(ctx).Close(ctx, conn)
}
