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
	"github.com/stretchr/testify/require"

	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/chainstest"
)

func TestNSMGR_RemoteUsecase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	domain, supplier := chainstest.NewDomainBuilder(t).
		SetNodesCount(2).
		SetContext(ctx).
		Build()
	defer supplier.Cleanup()

	nsc := supplier.SupplyNSC("nsc-1", domain.Nodes[0].NSMgr.URL)
	supplier.SupplyNSE(&registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}, domain.Nodes[1].NSMgr)

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
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 8, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.GetConnection().Context = conn.Context
	refreshRequest.GetConnection().Mechanism = conn.Mechanism
	refreshRequest.GetConnection().NetworkServiceEndpointName = conn.NetworkServiceEndpointName

	var connection2 *networkservice.Connection
	connection2, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, connection2)
	require.Equal(t, 8, len(connection2.Path.PathSegments))
}

func TestNSMGR_LocalUsecase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	domain, supplier := chainstest.NewDomainBuilder(t).
		SetNodesCount(1).
		SetContext(ctx).
		Build()
	defer supplier.Cleanup()

	supplier.SupplyNSE(&registry.NetworkServiceEndpoint{
		Name:                "final-endpoint",
		NetworkServiceNames: []string{"my-service-remote"},
	}, domain.Nodes[0].NSMgr)

	nsc := supplier.SupplyNSC("nsc-1", domain.Nodes[0].NSMgr.URL)

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
	require.NoError(t, err)
	require.NotNil(t, conn)

	require.Equal(t, 5, len(conn.Path.PathSegments))

	// Simulate refresh from client.

	refreshRequest := request.Clone()
	refreshRequest.GetConnection().Context = conn.Context
	refreshRequest.GetConnection().Mechanism = conn.Mechanism
	refreshRequest.GetConnection().NetworkServiceEndpointName = conn.NetworkServiceEndpointName

	var connection2 *networkservice.Connection
	connection2, err = nsc.Request(ctx, refreshRequest)
	require.NoError(t, err)
	require.NotNil(t, connection2)
	require.Equal(t, 5, len(connection2.Path.PathSegments))
}
