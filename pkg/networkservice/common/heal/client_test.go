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

package heal_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/heal"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatepath"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/updatetoken"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/eventchannel"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

type testOnHeal struct {
	RequestFunc func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error)
	CloseFunc   func(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error)
}

func (t *testOnHeal) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.RequestFunc(ctx, in, opts...)
}

func (t *testOnHeal) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	return t.CloseFunc(ctx, in, opts...)
}

func TestHealClient_Request(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	eventCh := make(chan *networkservice.ConnectionEvent, 1)
	defer close(eventCh)

	onHealCh := make(chan struct{})
	// TODO for tomorrow... check on how to work onHeal into the new chain I've built
	onHeal := &testOnHeal{
		RequestFunc: func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (connection *networkservice.Connection, e error) {
			if ctx.Err() == nil {
				close(onHealCh)
			}
			return &networkservice.Connection{}, nil
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	monitorServer := eventchannel.NewMonitorServer(eventCh)
	server := chain.NewNetworkServiceServer(
		updatepath.NewServer("testServer"),
		monitor.NewServer(ctx, &monitorServer),
		updatetoken.NewServer(sandbox.GenerateTestToken),
	)
	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("testClient"),
		heal.NewClient(ctx, adapters.NewMonitorServerToClient(monitorServer), addressof.NetworkServiceClient(onHeal)),
		updatetoken.NewClient(sandbox.GenerateTestToken),
		adapters.NewServerToClient(server),
	)

	requestCtx, reqCancelFunc := context.WithTimeout(ctx, waitForTimeout)
	defer reqCancelFunc()
	conn, err := client.Request(requestCtx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "ns-1",
		},
	})
	require.Nil(t, err)
	t1 := time.Now()
	_, err = server.Close(requestCtx, conn.Clone())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), waitHealTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		require.FailNow(t, "timeout waiting for Heal event %v", time.Since(t1))
		return
	case <-onHealCh:
		// All is fine, test is passed
		break
	}
	_, err = client.Close(requestCtx, conn)
	require.NoError(t, err)
}

func TestHealClient_EmptyInit(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	eventCh := make(chan *networkservice.ConnectionEvent, 1)
	defer close(eventCh)

	onHealCh := make(chan struct{})
	// TODO for tomorrow... check on how to work onHeal into the new chain I've built
	onHeal := &testOnHeal{
		RequestFunc: func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (connection *networkservice.Connection, e error) {
			if ctx.Err() == nil {
				close(onHealCh)
			}
			return &networkservice.Connection{}, nil
		},
	}

	eventTrigger := &testOnHeal{
		RequestFunc: func(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
			eventCh <- &networkservice.ConnectionEvent{
				Type:        networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER,
				Connections: make(map[string]*networkservice.Connection),
			}
			return next.Client(ctx).Request(ctx, in)
		},
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("testClient"),
		heal.NewClient(ctx, eventchannel.NewMonitorConnectionClient(eventCh), addressof.NetworkServiceClient(onHeal)),
		updatetoken.NewClient(sandbox.GenerateTestToken),
		updatepath.NewClient("testServer"),
		eventTrigger,
		updatetoken.NewClient(sandbox.GenerateTestToken),
	)

	requestCtx, reqCancelFunc := context.WithTimeout(ctx, waitForTimeout)
	defer reqCancelFunc()

	_, err := client.Request(requestCtx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "ns-1",
		},
	})
	require.Error(t, err)
}
