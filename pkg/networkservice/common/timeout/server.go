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

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/mapexecutor"
)

type timeoutServer struct {
	connections map[string]*timer
	executor    mapexecutor.Executor
}

type timer struct {
	timer  *time.Timer
	stopCh chan struct{}
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections.
func NewServer() networkservice.NetworkServiceServer {
	return &timeoutServer{
		connections: map[string]*timer{},
	}
}

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "Request")

	connID := request.GetConnection().GetId()
	<-t.executor.AsyncExec(connID, func() {
		if timer, ok := t.connections[connID]; ok {
			if !timer.timer.Stop() {
				logEntry.Warnf("connection has been timed out, re requesting: %v", connID)
			}
			close(timer.stopCh)
			delete(t.connections, connID)
		}

		conn, err = next.Server(ctx).Request(ctx, request)
		if err != nil {
			return
		}

		var timer *timer
		timer, err = t.createTimer(ctx, conn)
		if err != nil {
			if _, closeErr := next.Server(ctx).Close(ctx, conn); closeErr != nil {
				err = errors.Wrapf(err, "error attempting to close failed connection %v: %+v", connID, closeErr)
			}
			return
		}

		t.connections[connID] = timer
	})

	return conn, err
}

func (t *timeoutServer) createTimer(ctx context.Context, conn *networkservice.Connection) (*timer, error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "createTimer")

	expireTime, err := ptypes.Timestamp(conn.GetPath().GetPathSegments()[conn.GetPath().GetIndex()-1].GetExpires())
	if err != nil {
		return nil, err
	}

	conn = conn.Clone()
	ctx = extend.WithValuesFromContext(context.Background(), ctx)

	timer := &timer{
		stopCh: make(chan struct{}, 1),
	}
	timer.timer = time.AfterFunc(time.Until(expireTime), func() {
		t.executor.AsyncExec(conn.GetId(), func() {
			select {
			case <-timer.stopCh:
				logEntry.Warnf("timer has been already stopped: %v", conn.GetId())
			default:
				if err := t.close(ctx, conn); err != nil {
					logEntry.Errorf("failed to close timed out connection: %v %+v", conn.GetId(), err)
				}
			}
		})
	})

	return timer, nil
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	<-t.executor.AsyncExec(conn.GetId(), func() {
		err = t.close(ctx, conn)
	})
	return &empty.Empty{}, err
}

func (t *timeoutServer) close(ctx context.Context, conn *networkservice.Connection) error {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "close")

	timer, ok := t.connections[conn.GetId()]
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", conn.GetId())
		return nil
	}

	timer.timer.Stop()
	close(timer.stopCh)
	delete(t.connections, conn.GetId())

	_, err := next.Server(ctx).Close(ctx, conn)
	return err
}
