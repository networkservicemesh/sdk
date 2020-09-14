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

// Package nsmgr_test define a tests for NSMGR chain element.
package nsmgr_test

import (
	"context"
	"testing"

	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/cls"
	"github.com/networkservicemesh/api/pkg/api/networkservice/mechanisms/kernel"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/stretchr/testify/suite"
)

func (t *NSMGRSuite) TestRemoteNSMGR() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	nsc := t.NewClient(context.Background(), t.Cluster().Nodes[1].NSMgrURL)

	t.NewEndpoint(ctx, &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}, t.Cluster().Nodes[0].NSMgr)

	var conn *networkservice.Connection

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service-remote",
			Context:        &networkservice.ConnectionContext{},
		},
	}
	conn, err := nsc.Request(ctx, request)
	t.NoError(err)
	t.NotNil(conn)

	t.Equal(8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.GetConnection().Context = conn.Context
	refreshRequest.GetConnection().Mechanism = conn.Mechanism
	refreshRequest.GetConnection().NetworkServiceEndpointName = conn.NetworkServiceEndpointName

	var connection2 *networkservice.Connection
	connection2, err = nsc.Request(ctx, refreshRequest)
	t.NoError(err)
	t.NotNil(connection2)
	t.Equal(8, len(connection2.Path.PathSegments))
}

func TestNSMGRSuite(t *testing.T) {
	suite.Run(t, new(NSMGRSuite))
}

func (t *NSMGRSuite) TestNSMGR_LocalCase() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	t.NewEndpoint(ctx, &registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service"},
	}, t.Cluster().Nodes[0].NSMgr)

	cl := t.NewClient(context.Background(), t.Cluster().Nodes[0].NSMgrURL)

	var connection *networkservice.Connection

	request := &networkservice.NetworkServiceRequest{
		MechanismPreferences: []*networkservice.Mechanism{
			{Cls: cls.LOCAL, Type: kernel.MECHANISM},
		},
		Connection: &networkservice.Connection{
			Id:             "1",
			NetworkService: "my-service",
			Context:        &networkservice.ConnectionContext{},
		},
	}
	connection, err := cl.Request(ctx, request)
	t.NoError(err)
	t.NotNil(connection)

	t.Equal(5, len(connection.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.GetConnection().Context = connection.Context
	refreshRequest.GetConnection().Mechanism = connection.Mechanism
	refreshRequest.GetConnection().NetworkServiceEndpointName = connection.NetworkServiceEndpointName

	var connection2 *networkservice.Connection
	connection2, err = cl.Request(ctx, refreshRequest)
	t.NoError(err)
	t.NotNil(connection2)
	t.Equal(5, len(connection2.Path.PathSegments))
}
