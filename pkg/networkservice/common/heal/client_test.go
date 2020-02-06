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
	. "github.com/networkservicemesh/sdk/pkg/test/fixtures/healclientfixture"
	"reflect"
	"testing"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/stretchr/testify/require"
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
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
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
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
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
