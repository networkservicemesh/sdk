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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/test/fakes/fakemonitorconnection"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"reflect"
	"testing"
	"time"
)

type clientRequest func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error)

type testOnHeal struct {
	r clientRequest
}

func (t *testOnHeal) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.r(ctx, in, opts...)
}

func (t *testOnHeal) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	panic("implement me")
}

// callNotifier sends struct{}{} to notifier channel when h called
func callNotifier(notifier chan struct{}, h clientRequest) clientRequest {
	return func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (i *networkservice.Connection, e error) {
		notifier <- struct{}{}
		if h != nil {
			return h(ctx, in, opts...)
		}
		return nil, nil
	}
}

func TestHealClient_Request(t *testing.T) {
	monitorServer := fakemonitorconnection.New()
	defer monitorServer.Close()

	monitorClient, err := monitorServer.Client(context.Background())
	require.Nil(t, err)

	onHeal := &testOnHeal{}

	healClient := heal.NewClient(monitorClient, addressof.NetworkServiceClient(onHeal)).(*heal.HealClient)
	healChain := chain.NewNetworkServiceClient(healClient)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
		},
	}
	conn, err := healChain.Request(context.Background(), request)
	defer func() { _, _ = healChain.Close(context.Background(), conn) }()

	require.True(t, reflect.DeepEqual(conn, request.GetConnection()))
	require.Nil(t, err)

	calledOnceNotifier := make(chan struct{})
	onHeal.r = callNotifier(calledOnceNotifier, onHeal.r)

	stream, closeStream, err := monitorServer.Stream(context.Background())
	require.Nil(t, err)
	defer closeStream()

	err = stream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	err = stream.Send(&networkservice.ConnectionEvent{
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
		case <-calledOnceNotifier:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
}

func TestHealClient_MonitorClose(t *testing.T) {
	monitorServer := fakemonitorconnection.New()
	defer monitorServer.Close()
	monitorClient, err := monitorServer.Client(context.Background())
	require.Nil(t, err)

	onHeal := &testOnHeal{}

	healClient := heal.NewClient(monitorClient, addressof.NetworkServiceClient(onHeal)).(*heal.HealClient)
	healChain := chain.NewNetworkServiceClient(healClient)

	request := &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			Id:             "conn-1",
			NetworkService: "ns-1",
		},
	}
	conn, err := healChain.Request(context.Background(), request)
	defer func() { _, _ = healChain.Close(context.Background(), conn) }()

	require.True(t, reflect.DeepEqual(conn, request.GetConnection()))
	require.Nil(t, err)

	calledOnceNotifier := make(chan struct{})
	onHeal.r = callNotifier(calledOnceNotifier, onHeal.r)

	stream, closeStream, err := monitorServer.Stream(context.Background())
	require.Nil(t, err)

	err = stream.Send(&networkservice.ConnectionEvent{
		Type: networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
		Connections: map[string]*networkservice.Connection{
			"conn-1": {
				Id:             "conn-1",
				NetworkService: "ns-1",
			},
		},
	})
	require.Nil(t, err)

	closeStream()

	cond := func() bool {
		select {
		case <-calledOnceNotifier:
			return true
		default:
			return false
		}
	}
	require.Eventually(t, cond, 5*time.Second, 10*time.Millisecond)
}
