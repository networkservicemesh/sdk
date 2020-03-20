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

package monitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/heal"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type monitorClient struct {
	grpcClient    networkservice.MonitorConnectionClient
	healer        heal.Healer
	eventReceiver networkservice.MonitorConnection_MonitorConnectionsClient
	monitoring    map[string]struct{}
	reported      map[string]*networkservice.Connection
	chainContext  context.Context
	executor      serialize.Executor
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that will handle MonitorConnectionClient events
//             - ctx    	- context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - grpcClient - networkservice.MonitorConnectionClient that can be used to call MonitorConnection
//            	 			  against the endpoint.
//			   - healer 	- heal.Healer that can be used to start healing of broken connections.
func NewClient(ctx context.Context, grpcClient networkservice.MonitorConnectionClient, healer heal.Healer) networkservice.NetworkServiceClient {
	mc := &monitorClient{
		grpcClient:    grpcClient,
		healer:        healer,
		eventReceiver: nil,
		monitoring:    make(map[string]struct{}),
		reported:      make(map[string]*networkservice.Connection),
		chainContext:  ctx,
		executor:      serialize.Executor{},
	}
	mc.init()
	return mc
}

func (m *monitorClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	m.executor.AsyncExec(func() {
		m.monitoring[rv.GetId()] = struct{}{}
	})
	return rv, nil
}

func (m *monitorClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	m.executor.AsyncExec(func() {
		delete(m.monitoring, conn.GetId())
	})
	return rv, nil
}

func (m *monitorClient) init() {
	if m.eventReceiver != nil {
		select {
		case <-m.eventReceiver.Context().Done():
			return
		default:
			m.eventReceiver = nil
		}
	}

	recv, err := m.grpcClient.MonitorConnections(m.chainContext, &networkservice.MonitorScopeSelector{}, grpc.WaitForReady(true))
	if err != nil {
		m.executor.AsyncExec(m.init)
		return
	}

	logrus.Info("Creating new eventReceiver")

	m.eventReceiver = recv
	m.executor.AsyncExec(m.recvEvent)
}

func (m *monitorClient) recvEvent() {
	select {
	case <-m.eventReceiver.Context().Done():
		return
	default:
		event, err := m.eventReceiver.Recv()
		m.executor.AsyncExec(func() {
			if err != nil {
				for id := range m.monitoring {
					m.healer.StartHeal(id)
					delete(m.reported, id)
				}
				m.init()
				return
			}

			switch event.GetType() {
			case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
				m.reported = event.GetConnections()
				if event.GetConnections() == nil {
					m.reported = make(map[string]*networkservice.Connection)
				}

			case networkservice.ConnectionEventType_UPDATE:
				for _, conn := range event.GetConnections() {
					m.reported[conn.GetId()] = conn
				}

			case networkservice.ConnectionEventType_DELETE:
				for _, conn := range event.GetConnections() {
					delete(m.reported, conn.GetId())
				}
			}

			for id := range m.monitoring {
				if _, ok := m.reported[id]; !ok {
					m.healer.StartHeal(id)
				}
			}
		})
	}
	m.executor.AsyncExec(m.recvEvent)
}
