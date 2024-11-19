// Copyright (c) 2021-2024 Cisco and/or its affiliates.
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

package begin

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/tools/log"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type beginServer struct {
	genericsync.Map[string, *eventFactoryServer]
}

// NewServer - creates a new begin chain elem ent
func NewServer() networkservice.NetworkServiceServer {
	return &beginServer{}
}

func (b *beginServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	// No connection.ID, no service
	if request.GetConnection().GetId() == "" {
		return nil, errors.New("request.EventFactory.Id must not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.Server(ctx).Request(ctx, request)
	}
	eventFactoryServer, _ := b.LoadOrStore(request.GetConnection().GetId(),
		newEventFactoryServer(
			ctx,
			func() {
				b.Delete(request.GetRequestConnection().GetId())
			},
		),
	)
	<-eventFactoryServer.executor.AsyncExec(func() {
		currentEventFactoryServer, _ := b.Load(request.GetConnection().GetId())
		if currentEventFactoryServer != eventFactoryServer {
			log.FromContext(ctx).Debug("recalling begin.Request because currentEventFactoryServer != eventFactoryServer")
			conn, err = b.Request(ctx, request)
			return
		}

		if eventFactoryServer.state == established &&
			request.GetConnection().GetState() == networkservice.State_RESELECT_REQUESTED &&
			eventFactoryServer.request != nil && eventFactoryServer.request.Connection != nil {
			log.FromContext(ctx).Info("Closing connection due to RESELECT_REQUESTED state")

			eventFactoryCtx, eventFactoryCtxCancel := eventFactoryServer.ctxFunc()
			_, closeErr := next.Server(eventFactoryCtx).Close(eventFactoryCtx, eventFactoryServer.request.Connection)
			if closeErr != nil {
				log.FromContext(ctx).Errorf("Can't close old connection: %v", closeErr)
			}
			eventFactoryServer.state = closed
			eventFactoryCtxCancel()
		}

		withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
		conn, err = next.Server(withEventFactoryCtx).Request(withEventFactoryCtx, request)
		if err != nil {
			if eventFactoryServer.state != established {
				eventFactoryServer.state = closed
				b.Delete(request.GetConnection().GetId())
			}
			return
		}
		conn.State = networkservice.State_UP
		eventFactoryServer.request = request.Clone()
		eventFactoryServer.request.Connection = conn.Clone()
		eventFactoryServer.state = established

		eventFactoryServer.returnedConnection = conn.Clone()
		eventFactoryServer.updateContext(ctx)
	})
	return conn, err
}

func (b *beginServer) Close(ctx context.Context, conn *networkservice.Connection) (emp *emptypb.Empty, err error) {
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.Server(ctx).Close(ctx, conn)
	}
	eventFactoryServer, ok := b.Load(conn.GetId())
	if !ok {
		// If we don't have a connection to Close, just let it be
		return &emptypb.Empty{}, nil
	}
	<-eventFactoryServer.executor.AsyncExec(func() {
		if eventFactoryServer.state != established || eventFactoryServer.request == nil {
			return
		}
		currentServerClient, _ := b.Load(conn.GetId())
		if currentServerClient != eventFactoryServer {
			return
		}
		// Always close with the last valid EventFactory we got
		conn = eventFactoryServer.request.Connection
		withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
		emp, err = next.Server(withEventFactoryCtx).Close(withEventFactoryCtx, conn)
		eventFactoryServer.afterCloseFunc()
	})
	return &emptypb.Empty{}, err
}
