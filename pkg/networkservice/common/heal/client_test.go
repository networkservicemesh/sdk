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

package heal_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	. "github.com/networkservicemesh/sdk/pkg/test/fixtures/healclientfixture"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
)

const (
	waitForTimeout = 5 * time.Second
	tickTimeout    = 10 * time.Millisecond
)

func TestHealClient_Request(t *testing.T) {
	f := &Fixture{
		Request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	}
	err := SetupWithSingleRequest(f)
	defer TearDown(f)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(f.Conn, f.Request.GetConnection()))

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	cond := func() bool {
		select {
		case <-f.OnHealNotifierCh:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, cond, waitForTimeout, tickTimeout)
}

func TestHealClient_MonitorClose(t *testing.T) {
	f := &Fixture{
		Request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	}
	err := SetupWithSingleRequest(f)
	defer TearDown(f)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(f.Conn, f.Request.GetConnection()))

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	f.CloseStream()

	cond := func() bool {
		select {
		case <-f.OnHealNotifierCh:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, cond, waitForTimeout, tickTimeout)
}

func TestHealClient_EmptyInit(t *testing.T) {
	f := &Fixture{
		Request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	}
	err := SetupWithSingleRequest(f)
	defer TearDown(f)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(f.Conn, f.Request.GetConnection()))

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: make(map[string]*networkservice.Connection),
	})
	require.Nil(t, err)

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)
}

func TestHealClient_SeveralConnection(t *testing.T) {
	f := &Fixture{
		Request: &networkservice.NetworkServiceRequest{
			Connection: &networkservice.Connection{
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	}
	err := SetupWithSingleRequest(f)
	defer TearDown(f)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(f.Conn, f.Request.GetConnection()))

	r := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-2",
			NetworkService: "ns-2",
		},
	}

	conn, err := f.Client.Request(context.Background(), r)
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(conn, r.GetConnection()))

	ctx, cancelFunc := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, _, err = f.MockMonitorServer.Stream(ctx)
	require.NotNil(t, err) // we expect healClient called MonitorConnections method once
	cancelFunc()
}

func TestNewClient_MissingConnectionsInInit(t *testing.T) {
	f := &Fixture{}
	err := Setup(f)
	defer TearDown(f)
	require.Nil(t, err)

	conns := []*networkservice.Connection{
		{Id: "conn-1", NetworkService: "ns-1"},
		{Id: "conn-2", NetworkService: "ns-2"},
	}

	conn, err := f.Client.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: conns[0]})
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(conn, conns[0]))

	conn, err = f.Client.Request(context.Background(), &networkservice.NetworkServiceRequest{Connection: conns[1]})
	require.Nil(t, err)
	require.True(t, reflect.DeepEqual(conn, conns[1]))

	err = f.ServerStream.Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{conns[0].GetId(): conns[0]},
	})
	require.Nil(t, err)

	// we emulate situation that server managed to handle only the first connection
	// second connection should came in the UPDATE event, but we emulate server's falling down
	f.CloseStream()
	// at that point we expect that 'healClient' start healing both 'conn-1' and 'conn-2'

	healedIDs := map[string]bool{}

	cond := func() bool {
		select {
		case r := <-f.OnHealNotifierCh:
			if _, ok := healedIDs[r.GetConnection().GetId()]; !ok {
				healedIDs[r.GetConnection().GetId()] = true
				return true
			}
			return false
		default:
			return false
		}
	}

	require.Eventually(t, cond, waitForTimeout, tickTimeout)
	require.Eventually(t, cond, waitForTimeout, tickTimeout)

	require.True(t, healedIDs[conns[0].GetId()])
	require.True(t, healedIDs[conns[1].GetId()])
}
