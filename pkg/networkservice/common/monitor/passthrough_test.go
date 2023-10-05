// Copyright (c) 2021-2023 Cisco and/or its affiliates.
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
	"fmt"
	"net/url"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/suite"
	"go.uber.org/goleak"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

const (
	passThoughCount = 10
)

type MonitorPassThroughSuite struct {
	suite.Suite
	testCtx    context.Context
	testCancel context.CancelFunc

	passThroughEndpoints     []endpoint.Endpoint
	passThroughListenOnURLs  []*url.URL
	passThroughConnectToURLs []*url.URL
	passThroughCtxs          []context.Context
	passThroughCancels       []context.CancelFunc

	endpoint       endpoint.Endpoint
	endpointURL    *url.URL
	endpointCtx    context.Context
	endpointCancel context.CancelFunc

	client networkservice.NetworkServiceClient

	monitorClient networkservice.MonitorConnection_MonitorConnectionsClient
	conn          *networkservice.Connection
}

func (m *MonitorPassThroughSuite) SetupTest() {
	m.T().Cleanup(func() { goleak.VerifyNone(m.T()) })
	m.testCtx, m.testCancel = context.WithCancel(context.Background())

	m.StartEndPoint()
	m.StartPassThroughEndpoints(m.endpointURL)
	m.StartClient(m.passThroughListenOnURLs[0])
	m.StartMonitor(m.passThroughListenOnURLs[0])

	event, err := m.monitorClient.Recv()
	m.Require().NoError(err)
	m.Require().NotNil(event)
	m.Require().Equal(networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, event.GetType())
	m.Require().Nil(event.GetConnections())

	m.conn, err = m.client.Request(m.testCtx, &networkservice.NetworkServiceRequest{})
	m.Assert().NoError(err)
	m.Assert().NotNil(m.conn)
	m.Assert().NotNil(m.conn.GetPath())
	m.Require().Equal(m.conn.GetPath().GetIndex(), uint32(0))
	m.Require().Len(m.conn.GetPath().GetPathSegments(), len(m.passThroughEndpoints)+2)

	event, err = m.monitorClient.Recv()
	m.Require().NoError(err)
	m.Require().NotNil(event)
	m.Require().Equal(networkservice.ConnectionEventType_UPDATE, event.GetType())
	m.Require().NotNil(event.GetConnections())
	expectedConn := m.conn.Clone()
	expectedConn.GetPath().Index++
	expectedConn.Id = expectedConn.GetCurrentPathSegment().GetId()
	actualConn := event.GetConnections()[expectedConn.GetId()]
	m.Require().True(proto.Equal(expectedConn, actualConn))
}

func (m *MonitorPassThroughSuite) TearDownTest() {
	m.testCancel()
}

func (m *MonitorPassThroughSuite) StartPassThroughEndpoints(connectTo *url.URL) {
	m.passThroughCtxs = nil
	m.passThroughCancels = nil
	m.passThroughListenOnURLs = nil
	m.passThroughConnectToURLs = nil
	m.passThroughEndpoints = nil
	for len(m.passThroughCtxs) < passThoughCount {
		passThroughCtx, passThroughCancel := context.WithCancel(m.testCtx)
		passThroughListenOnURL := &url.URL{Scheme: "tcp", Host: "127.0.0.1:"}
		name := fmt.Sprintf("passthrough-%d", len(m.passThroughCtxs))
		passThroughConnectToURL := &url.URL{}
		passThroughEndpoint := endpoint.NewServer(
			passThroughCtx,
			sandbox.GenerateExpiringToken(time.Second),
			endpoint.WithName(name),
			endpoint.WithAuthorizeServer(null.NewServer()),
			endpoint.WithAdditionalFunctionality(
				clienturl.NewServer(passThroughConnectToURL),
				connect.NewServer(
					client.NewClient(
						passThroughCtx,
						client.WithName(name),
						client.WithoutRefresh(),
						client.WithAuthorizeClient(null.NewClient()),
						client.WithDialTimeout(sandbox.DialTimeout),
						client.WithDialOptions(
							sandbox.DialOptions()...,
						),
					),
				),
			),
		)
		m.Require().NoError(startEndpoint(passThroughCtx, passThroughListenOnURL, passThroughEndpoint))
		if len(m.passThroughConnectToURLs)-1 >= 0 {
			*(m.passThroughConnectToURLs[len(m.passThroughConnectToURLs)-1]) = *passThroughListenOnURL
		}
		m.passThroughCtxs = append(m.passThroughCtxs, passThroughCtx)
		m.passThroughCancels = append(m.passThroughCancels, passThroughCancel)
		m.passThroughListenOnURLs = append(m.passThroughListenOnURLs, passThroughListenOnURL)
		m.passThroughConnectToURLs = append(m.passThroughConnectToURLs, passThroughConnectToURL)
		m.passThroughEndpoints = append(m.passThroughEndpoints, passThroughEndpoint)
	}
	if len(m.passThroughConnectToURLs)-1 >= 0 {
		*(m.passThroughConnectToURLs[len(m.passThroughConnectToURLs)-1]) = *connectTo
	}
}

func (m *MonitorPassThroughSuite) StartEndPoint() {
	m.endpointCtx, m.endpointCancel = context.WithCancel(m.testCtx)
	m.endpointURL = &url.URL{Scheme: "tcp", Host: "127.0.0.1:"}
	name := "endpoint"
	m.endpoint = endpoint.NewServer(
		m.testCtx,
		sandbox.GenerateExpiringToken(time.Second),
		endpoint.WithName(name),
		endpoint.WithAuthorizeServer(null.NewServer()),
	)
	m.Require().NoError(startEndpoint(m.endpointCtx, m.endpointURL, m.endpoint))
}

func (m *MonitorPassThroughSuite) StartClient(connectTo *url.URL) {
	m.client = client.NewClient(
		m.testCtx,
		client.WithClientURL(connectTo),
		client.WithAuthorizeClient(null.NewClient()),
		client.WithDialTimeout(sandbox.DialTimeout),
		client.WithDialOptions(
			sandbox.DialOptions()...,
		),
	)
}

func (m *MonitorPassThroughSuite) StartMonitor(connectTo *url.URL) {
	target := grpcutils.URLToTarget(connectTo)
	cc, err := grpc.DialContext(m.testCtx, target, grpc.WithBlock(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	m.Require().NoError(err)
	m.Require().NotNil(cc)
	go func(ctx context.Context, cc *grpc.ClientConn) {
		<-ctx.Done()
		_ = cc.Close()
	}(m.testCtx, cc)
	c := networkservice.NewMonitorConnectionClient(cc)
	mc, err := c.MonitorConnections(m.testCtx, &networkservice.MonitorScopeSelector{})
	m.Require().NoError(err)
	m.Require().NotNil(mc)
	m.monitorClient = mc
}

func (m *MonitorPassThroughSuite) ValidateEvent(expectedType networkservice.ConnectionEventType, expectedConn *networkservice.Connection) {
	event, err := m.monitorClient.Recv()
	m.Require().NoError(err)
	m.Require().NotNil(event)
	m.Require().Equal(expectedType, event.GetType())
	m.Require().NotNil(event.GetConnections())
	nextID := expectedConn.GetNextPathSegment().GetId()
	actualConn := event.GetConnections()[nextID]
	m.Require().Equal(nextID, actualConn.GetId())
	m.Require().Equal(len(expectedConn.GetPath().GetPathSegments()), len(actualConn.GetPath().GetPathSegments()))
	for i, actualSegment := range actualConn.GetPath().GetPathSegments() {
		expectedSegment := expectedConn.GetPath().GetPathSegments()[i]
		m.Assert().Equal(expectedSegment.GetId(), actualSegment.GetId())
		m.Assert().Equal(expectedSegment.GetName(), actualSegment.GetName())
	}
	m.Require().Equal(expectedConn.GetContext(), actualConn.GetContext())
	m.Require().Equal(expectedConn.GetPath().GetIndex()+1, actualConn.GetPath().GetIndex())
	m.Require().Equal(expectedConn.GetState(), actualConn.GetState())
}

func (m *MonitorPassThroughSuite) TestDeleteToDown() {
	// Have the endpoint close the connection
	closeConn := m.conn.Clone()
	closeConn.GetPath().Index = uint32(len(closeConn.GetPath().GetPathSegments()) - 1)
	closeConn.Id = closeConn.GetCurrentPathSegment().GetId()
	_, err := m.endpoint.Close(m.testCtx, closeConn)
	m.Require().NoError(err)
	expectedConn := m.conn.Clone()
	expectedConn.State = networkservice.State_DOWN
	m.ValidateEvent(networkservice.ConnectionEventType_UPDATE, expectedConn)
}

func (m *MonitorPassThroughSuite) TestServerDown() {
	// Disconnect the Server
	m.endpointCancel()
	m.Require().NoError(waitServerStopped(m.endpointURL))

	expectedConn := m.conn.Clone()
	expectedConn.State = networkservice.State_DOWN
	m.ValidateEvent(networkservice.ConnectionEventType_UPDATE, expectedConn)
}

func (m *MonitorPassThroughSuite) TestServerUpdate() {
	// Have the endpoint close the connection
	endpointConn := m.conn.Clone()
	endpointConn.GetPath().Index = uint32(len(endpointConn.GetPath().GetPathSegments()) - 1)
	endpointConn.Id = endpointConn.GetCurrentPathSegment().GetId()
	endpointConn.State = networkservice.State_DOWN
	endpointConn.Context = &networkservice.ConnectionContext{
		ExtraContext: map[string]string{"mark": "true"},
	}
	_, err := m.endpoint.Request(m.testCtx, &networkservice.NetworkServiceRequest{
		Connection: endpointConn.Clone(),
	})
	m.Require().NoError(err)

	expectedConn := endpointConn.Clone()
	expectedConn.GetPath().Index = m.conn.GetPath().GetIndex()
	expectedConn.Id = expectedConn.GetCurrentPathSegment().GetId()
	m.ValidateEvent(networkservice.ConnectionEventType_UPDATE, expectedConn)
}

func (m *MonitorPassThroughSuite) TestDeleteOnClientClose() {
	// Have the client close the connection
	_, err := m.client.Close(m.testCtx, m.conn)
	m.Require().NoError(err)
	m.ValidateEvent(networkservice.ConnectionEventType_DELETE, m.conn)
}

func TestPassThrough(t *testing.T) {
	suite.Run(t, &MonitorPassThroughSuite{})
}
