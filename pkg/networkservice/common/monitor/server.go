// Copyright (c) 2020-2022 Cisco Systems, Inc.
//
// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

// Package monitor provides a NetworkServiceServer chain element to provide a monitor server that reflects
// the connections actually in the NetworkServiceServer
package monitor

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clientconn"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"
	"github.com/networkservicemesh/sdk/pkg/tools/postpone"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type monitorServer struct {
	chainCtx context.Context
	networkservice.MonitorConnectionServer
}

// NewServer - creates a NetworkServiceServer chain element that will properly update a MonitorConnectionServer
//   - monitorServerPtr - *networkservice.MonitorConnectionServer.  Since networkservice.MonitorConnectionServer is an interface
//     (and thus a pointer) *networkservice.MonitorConnectionServer is a double pointer.  Meaning it
//     points to a place that points to a place that implements networkservice.MonitorConnectionServer
//     This is done so that we can preserve the return of networkservice.NetworkServer and use
//     NewServer(...) as any other chain element constructor, but also get back a
//     networkservice.MonitorConnectionServer that can be used either standalone or in a
//     networkservice.MonitorConnectionServer chain
//     chainCtx - context for lifecycle management
func NewServer(chainCtx context.Context, monitorServerPtr *networkservice.MonitorConnectionServer) networkservice.NetworkServiceServer {
	*monitorServerPtr = newMonitorConnectionServer(chainCtx)
	return &monitorServer{
		chainCtx:                chainCtx,
		MonitorConnectionServer: *monitorServerPtr,
	}
}

func (m *monitorServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	closeCtxFunc := postpone.ContextWithValues(ctx)
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelEventLoop()
	}

	storeEventConsumer(ctx, metadata.IsClient(m), m.MonitorConnectionServer.(EventConsumer))

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}
	_ = m.MonitorConnectionServer.(EventConsumer).Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_UPDATE,
		Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
	})

	// If we have a clientconn ... we must be part of a passthrough server, and have a client to pass
	// events through from, so start an eventLoop
	cc, ccLoaded := clientconn.Load(ctx)
	if ccLoaded {
		cancelEventLoop, eventLoopErr := newEventLoop(m.chainCtx, m.MonitorConnectionServer.(EventConsumer), cc, conn)
		if eventLoopErr != nil {
			closeCtx, closeCancel := closeCtxFunc()
			defer closeCancel()
			_, _ = next.Client(closeCtx).Close(closeCtx, conn)
			return nil, errors.Wrap(eventLoopErr, "unable to monitor")
		}
		store(ctx, metadata.IsClient(m), cancelEventLoop)
	}

	return conn, nil
}

func (m *monitorServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Cancel any existing eventLoop
	if cancelEventLoop, loaded := loadAndDelete(ctx, metadata.IsClient(m)); loaded {
		cancelEventLoop()
	}
	rv, err := next.Server(ctx).Close(ctx, conn)
	_ = m.MonitorConnectionServer.(EventConsumer).Send(&networkservice.ConnectionEvent{
		Type:        networkservice.ConnectionEventType_DELETE,
		Connections: map[string]*networkservice.Connection{conn.GetId(): conn.Clone()},
	})
	return rv, err
}
