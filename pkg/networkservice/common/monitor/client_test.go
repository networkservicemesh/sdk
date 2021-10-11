// Copyright (c) 2021 Cisco and/or its affiliates.
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

package monitor_test

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/timeout"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	testTimeout = time.Second
	pause       = 10 * time.Millisecond
)

type monitorClientSuite struct {
	suite.Suite
	conn         *networkservice.Connection
	client       networkservice.NetworkServiceClient
	server       endpoint.Endpoint
	ch           chan *networkservice.ConnectionEvent
	testCtx      context.Context
	testCancel   context.CancelFunc
	serverCancel context.CancelFunc
	url          *url.URL
}

func (m *monitorClientSuite) SetupTest() {
	m.T().Cleanup(func() { goleak.VerifyNone(m.T()) })
	m.testCtx, m.testCancel = context.WithCancel(context.Background())

	// Insert an event consumer into the context
	ec := newEventConsumer()
	m.testCtx = monitor.WithEventConsumer(m.testCtx, ec)

	tokengen := sandbox.GenerateExpiringToken(time.Second)

	// Create the server and serve
	m.url = &url.URL{Scheme: "tcp", Host: "127.0.0.1:"}
	m.server = endpoint.NewServer(
		m.testCtx,
		tokengen,
		endpoint.WithAuthorizeServer(null.NewServer()),
	)
	var serverCtx context.Context
	serverCtx, m.serverCancel = context.WithCancel(m.testCtx)
	m.Require().NoError(startServer(serverCtx, m.url, m.server))

	// Create the client
	m.client = client.NewClient(
		m.testCtx,
		client.WithClientURL(m.url),
		client.WithAuthorizeClient(null.NewClient()),
		client.WithDialOptions(
			sandbox.DialOptions()...,
		),
	)
	// Make a request
	var err error
	m.conn, err = m.client.Request(m.testCtx, &networkservice.NetworkServiceRequest{})
	m.Assert().NoError(err)
	m.Assert().NotNil(m.conn)
	m.Assert().NotNil(m.conn.GetPath())
	m.Require().Equal(m.conn.GetPath().GetIndex(), uint32(0))
	m.Require().Len(m.conn.GetPath().GetPathSegments(), 2)

	select {
	case event := <-ec.ch:
		m.Assert().Fail("should not have received event: %+v", event)
	case <-time.After(pause):
	}
	m.ch = ec.ch
}

func (m *monitorClientSuite) TearDownTest() {
	m.testCancel()
}

func (m *monitorClientSuite) TestDeleteToDown() {
	// Have the server close the connection
	closeConn := m.conn.Clone()
	closeConn.Id = closeConn.GetNextPathSegment().GetId()
	closeConn.GetPath().Index++
	_, err := m.server.Close(m.testCtx, closeConn)
	m.Require().NoError(err)

	select {
	case event := <-m.ch:
		m.Require().NotNil(event)
		m.Require().Len(event.GetConnections(), 1)
		m.Require().Equal(m.conn.GetId(), event.GetConnections()[m.conn.GetId()].GetId())
		m.Require().Equal(event.GetType(), networkservice.ConnectionEventType_UPDATE)
		m.Require().Equal(event.GetConnections()[m.conn.GetId()].GetState(), networkservice.State_DOWN)
	case <-time.After(testTimeout):
		m.Assert().Fail("testTimeout waiting for event")
	}
}

func (m *monitorClientSuite) TestServerDown() {
	// Disconnect the Server
	m.serverCancel()
	m.Require().NoError(waitServerStopped(m.url))

	select {
	case event := <-m.ch:
		m.Require().NotNil(event)
		m.Require().Len(event.GetConnections(), 1)
		m.Require().Equal(m.conn.GetId(), event.GetConnections()[m.conn.GetId()].GetId())
		m.Require().Equal(event.GetType(), networkservice.ConnectionEventType_UPDATE)
		m.Require().Equal(event.GetConnections()[m.conn.GetId()].GetState(), networkservice.State_DOWN)
	case <-time.After(testTimeout):
		m.Assert().Fail("testTimeout waiting for event")
	}
}

func (m *monitorClientSuite) TestServerUpdate() {
	// Have the server close the connection
	serverConn := m.conn.Clone()
	serverConn.Id = serverConn.GetNextPathSegment().GetId()
	serverConn.GetPath().Index++
	serverConn.State = networkservice.State_DOWN
	serverConn.Context = &networkservice.ConnectionContext{
		ExtraContext: map[string]string{"mark": "true"},
	}
	_, err := m.server.Request(m.testCtx, &networkservice.NetworkServiceRequest{
		Connection: serverConn,
	})
	m.Require().NoError(err)

	select {
	case event := <-m.ch:
		m.Require().NotNil(event)
		m.Require().Len(event.GetConnections(), 1)
		m.Require().Equal(m.conn.GetId(), event.GetConnections()[m.conn.GetId()].GetId())
		m.Require().Equal(event.GetType(), networkservice.ConnectionEventType_UPDATE)
		m.Require().Equal(event.GetConnections()[m.conn.GetId()].GetState(), networkservice.State_DOWN)
		m.Require().NotNil(event.GetConnections()[m.conn.GetId()].GetContext().GetExtraContext())
		m.Require().Equal(event.GetConnections()[m.conn.GetId()].GetContext().GetExtraContext()["mark"], "true")
	case <-time.After(testTimeout):
		m.Assert().Fail("testTimeout waiting for event")
	}
}

func (m *monitorClientSuite) TestNoEventOnClientClose() {
	// Have the client close the connection
	_, err := m.client.Close(m.testCtx, m.conn)
	m.Require().NoError(err)

	select {
	case event := <-m.ch:
		m.Assert().Fail("should not have received event: %+v", event)
	case <-time.After(pause):
	}
}

func TestMonitorClient(t *testing.T) {
	suite.Run(t, &monitorClientSuite{})
}

func TestMonitorClientError(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })
	testCtx, testCancel := context.WithCancel(context.Background())
	defer testCancel()

	// Insert an event consumer into the context
	ec := newEventConsumer()
	testCtx = monitor.WithEventConsumer(testCtx, ec)

	tokengen := sandbox.GenerateExpiringToken(time.Second)

	// Create the server and serve
	listenOn := &url.URL{Scheme: "tcp", Host: "127.0.0.1:"}
	name := "error-test"
	capture := &captureServer{}
	server := chain.NewNamedNetworkServiceServer(
		name,
		updatepath.NewServer(name),
		begin.NewServer(),
		updatetoken.NewServer(tokengen),
		metadata.NewServer(),
		timeout.NewServer(testCtx),
		capture,
	)

	// Serve only NetworkServiceMesh not MonitorConnection
	serverCtx, serverCancel := context.WithCancel(testCtx)
	defer serverCancel()
	grpcServer := grpc.NewServer()
	networkservice.RegisterNetworkServiceServer(grpcServer, server)
	grpcutils.RegisterHealthServices(grpcServer, server)

	errCh := grpcutils.ListenAndServe(serverCtx, listenOn, grpcServer)
	require.Len(t, errCh, 0)

	require.NoError(t, waitServerStarted(listenOn))

	// Create the client
	c := client.NewClient(
		testCtx,
		client.WithClientURL(listenOn),
		client.WithAuthorizeClient(null.NewClient()),
		client.WithDialOptions(
			sandbox.DialOptions()...,
		),
	)
	// Make a request - should fail due to lack of monitor server
	conn, err := c.Request(testCtx, &networkservice.NetworkServiceRequest{})
	assert.Error(t, err)
	assert.Nil(t, conn)

	assert.Nil(t, capture.capturedRequest)
}

type captureServer struct {
	capturedRequest *networkservice.NetworkServiceRequest
}

func (s *captureServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	s.capturedRequest = request.Clone()
	return next.Server(ctx).Request(ctx, request)
}

func (s *captureServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	s.capturedRequest = nil
	return next.Server(ctx).Close(ctx, conn)
}
