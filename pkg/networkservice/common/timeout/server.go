// Copyright (c) 2020 Cisco Systems, Inc.
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

// Package timeout provides a NetworkServiceServer chain element that times out expired connection
package timeout

import (
	"context"
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
)

type timeoutServer struct {
	connections map[string]*time.Timer
	executor    serialize.Executor
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections.
func NewServer() networkservice.NetworkServiceServer {
	return &timeoutServer{
		connections: make(map[string]*time.Timer),
	}
}

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	ct, err := t.createTimer(ctx, conn)
	if err != nil {
		if _, closeErr := next.Server(ctx).Close(ctx, conn); closeErr != nil {
			return nil, errors.Wrapf(err, "Error attempting to close failed connection %v: %+v", conn.GetId(), closeErr)
		}
		return nil, err
	}

	connID := conn.GetId()
	t.executor.AsyncExec(func() {
		if timer, ok := t.connections[connID]; !ok || timer.Stop() {
			t.connections[connID] = ct
		}
	})

	return conn, nil
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	connID := conn.GetId()
	t.executor.AsyncExec(func() {
		if timer, ok := t.connections[connID]; ok {
			timer.Stop()
			delete(t.connections, connID)
		}
	})
	return next.Server(ctx).Close(ctx, conn)
}

func (t *timeoutServer) createTimer(ctx context.Context, conn *networkservice.Connection) (*time.Timer, error) {
	conn = conn.Clone()
	ctx = extend.WithValuesFromContext(context.Background(), ctx)

	expireTime, err := ptypes.Timestamp(conn.GetPath().GetPathSegments()[conn.GetPath().GetIndex()-1].GetExpires())
	if err != nil {
		return nil, err
	}

	duration := time.Until(expireTime)
	return time.AfterFunc(duration, func() {
		if _, err := t.Close(ctx, conn); err != nil {
			trace.Log(ctx).Errorf("Error attempting to close timed out connection: %s: %+v", conn.GetId(), err)
		}
	}), nil
}
