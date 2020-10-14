// Copyright (c) 2020 Doc.ai and/or its affiliates.
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
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	kernelmech "github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/kernel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const (
	tokenTimeout = 100 * time.Millisecond
	closeTimeout = 10 * tokenTimeout
)

func TestTimeoutServer_Request(t *testing.T) {
	connServer := &connectionsServer{
		connections: map[string]*connectionInfo{},
	}

	server := struct {
		networkservice.NetworkServiceServer
	}{}
	server.NetworkServiceServer = chain.NewNetworkServiceServer(
		updatepath.NewServer("server"),
		timeout.NewServer(&server.NetworkServiceServer),
		mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
			kernelmech.MECHANISM: connServer,
		}),
	)

	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("client"),
		updatetoken.NewClient(func(_ credentials.AuthInfo) (string, time.Time, error) {
			return "token", time.Now().Add(tokenTimeout), nil
		}),
		kernel.NewClient(),
		adapters.NewServerToClient(
			server,
		),
	)

	_, err := client.Request(context.TODO(), &networkservice.NetworkServiceRequest{})
	require.NoError(t, err)
	require.True(t, connServer.validate(t, 1, 0))

	require.Eventually(t, func() bool {
		return connServer.validate(t, 0, 1)
	}, closeTimeout, 2*tokenTimeout)
}

type connectionsServer struct {
	lock        sync.Mutex
	connections map[string]*connectionInfo
}

type connectionInfo struct {
	requestCount int
	closeCount   int
}

func (s *connectionsServer) validate(t *testing.T, open, closed int) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	var connsOpen, connsClosed int
	for connID, connInfo := range s.connections {
		switch delta := connInfo.requestCount - connInfo.closeCount; {
		case delta > 1:
			require.Fail(t, "connection is double requested: %v", connID)
		case delta == 1:
			connsOpen++
		case delta == 0:
			connsClosed++
		case delta < 0:
			require.Fail(t, "connection is double closed %v", connID)
		}
	}

	if connsOpen != open {
		logrus.Warnf("open count is not equal: expected %v != actual %v", open, connsOpen)
		return false
	}
	if connsClosed != closed {
		logrus.Warnf("closed count is not equal: expected %v != actual %v", closed, connsClosed)
		return false
	}
	return true
}

func (s *connectionsServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.lock.Lock()

	connID := request.GetConnection().GetId()
	if _, ok := s.connections[connID]; !ok {
		s.connections[connID] = &connectionInfo{}
	}
	s.connections[connID].requestCount++

	s.lock.Unlock()

	return next.Server(ctx).Request(ctx, request)
}

func (s *connectionsServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.lock.Lock()

	connID := conn.GetId()
	if _, ok := s.connections[connID]; !ok {
		s.connections[connID] = &connectionInfo{}
	}
	s.connections[connID].closeCount++

	s.lock.Unlock()

	return next.Server(ctx).Close(ctx, conn)
}
