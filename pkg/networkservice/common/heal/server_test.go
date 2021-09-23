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
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/sandbox"
)

func TestHealClient_Request(t *testing.T) {
	t.Cleanup(func() { goleak.VerifyNone(t) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	onHealCh := make(chan struct{})
	// TODO for tomorrow... check on how to work onHeal into the new chain I've built
	var onHeal networkservice.NetworkServiceClient = &testOnHeal{
		requestFunc: func(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
			if err = ctx.Err(); err == nil {
				close(onHealCh)
			}
			return request.Connection, err
		},
	}

	eventCh := make(chan *networkservice.ConnectionEvent, 1)
	defer close(eventCh)

	monitorServer := eventchannel.NewMonitorServer(eventCh)

	server := chain.NewNetworkServiceServer(
		updatepath.NewServer("testServer"),
		monitor.NewServer(ctx, &monitorServer),
		updatetoken.NewServer(sandbox.GenerateTestToken),
	)

	client := chain.NewNetworkServiceClient(
		updatepath.NewClient("testClient"),
		metadata.NewClient(),
		adapters.NewServerToClient(heal.NewServer(ctx,
			heal.WithOnHeal(&onHeal))),
		heal.NewClient(ctx, adapters.NewMonitorServerToClient(monitorServer)),
		adapters.NewServerToClient(
			chain.NewNetworkServiceServer(
				updatetoken.NewServer(sandbox.GenerateTestToken),
				server,
			),
		),
	)

	conn, err := client.Request(ctx, &networkservice.NetworkServiceRequest{
		Connection: &networkservice.Connection{
			NetworkService: "ns-1",
		},
	})
	require.NoError(t, err)

	_, err = server.Close(ctx, conn.Clone())
	require.NoError(t, err)

	select {
	case <-ctx.Done():
		require.FailNow(t, "timeout waiting for Heal event")
	case <-onHealCh:
		// All is fine, test is passed
	}

	_, err = client.Close(ctx, conn)
	require.NoError(t, err)
}

type testOnHeal struct {
	requestFunc func(context.Context, *networkservice.NetworkServiceRequest, ...grpc.CallOption) (*networkservice.Connection, error)
}

func (t *testOnHeal) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	return t.requestFunc(ctx, request, opts...)
}

func (t *testOnHeal) Close(context.Context, *networkservice.Connection, ...grpc.CallOption) (*empty.Empty, error) {
	return new(empty.Empty), nil
}
