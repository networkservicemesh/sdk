// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package vl3mtu

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/monitor"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

const (
	jumboFrameSize = 9000
)

type vl3MtuServer struct {
	connections map[string]*networkservice.Connection
	executor    serialize.Executor
	minMtu      uint32
}

// NewServer - returns a new vl3mtu server chain element.
// It stores a minimum mtu of the vl3 and sends connectionEvent if refresh required.
func NewServer() networkservice.NetworkServiceServer {
	return &vl3MtuServer{
		minMtu:      jumboFrameSize,
		connections: make(map[string]*networkservice.Connection),
	}
}

func (v *vl3MtuServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Check MTU of the connection
	if request.GetConnection().GetContext() == nil {
		request.GetConnection().Context = &networkservice.ConnectionContext{}
	}
	<-v.executor.AsyncExec(func() {
		if request.GetConnection().GetContext().GetMTU() > v.minMtu || request.GetConnection().GetContext().GetMTU() == 0 {
			request.GetConnection().GetContext().MTU = v.minMtu
		}
	})

	ret, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return ret, err
	}

	conn := ret.Clone()
	v.executor.AsyncExec(func() {
		// We need to update minimum mtu of the vl3 network and send notifications to the already connected clients.
		logger := log.FromContext(ctx).WithField("vl3MtuServer", "Request")
		if conn.GetContext().GetMTU() < v.minMtu {
			v.minMtu = conn.GetContext().GetMTU()
			logger.Debug("MTU was updated")

			connections := make(map[string]*networkservice.Connection)
			for _, c := range v.connections {
				if c.GetId() == conn.GetId() {
					continue
				}
				connections[c.GetId()] = c.Clone()
				connections[c.GetId()].GetContext().MTU = v.minMtu
				connections[c.GetId()].State = networkservice.State_REFRESH_REQUESTED
			}
			if eventConsumer, ok := monitor.LoadEventConsumer(ctx, metadata.IsClient(v)); ok {
				_ = eventConsumer.Send(&networkservice.ConnectionEvent{
					Type:        networkservice.ConnectionEventType_UPDATE,
					Connections: connections,
				})
			} else {
				logger.Debug("eventConsumer is not presented")
			}
		}
		v.connections[conn.GetId()] = conn
	})

	return ret, nil
}

func (v *vl3MtuServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	v.executor.AsyncExec(func() {
		delete(v.connections, conn.GetId())
	})
	return next.Server(ctx).Close(ctx, conn)
}
