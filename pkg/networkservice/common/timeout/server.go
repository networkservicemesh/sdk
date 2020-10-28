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
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type timeoutServer struct {
	ctx         context.Context
	connections timerMap
	executors   executorMap
}

type timer struct {
	timer  *time.Timer
	stopCh chan struct{}
}

// NewServer - creates a new NetworkServiceServer chain element that implements timeout of expired connections.
func NewServer(ctx context.Context) networkservice.NetworkServiceServer {
	return &timeoutServer{
		ctx: ctx,
	}
}

func (t *timeoutServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (conn *networkservice.Connection, err error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "Request")

	connID := request.GetConnection().GetId()

	executor, _ := t.executors.LoadOrStore(connID, &serialize.Executor{})
	<-executor.AsyncExec(func() {
		// executor was possibly removed by `t.close()` at this moment, we need to store it back
		exec, _ := t.executors.LoadOrStore(connID, executor)
		if exec != executor {
			err = errors.Errorf("race condition, parallel request execution: %v", connID)
			return
		}

		if timer, ok := t.connections.Load(connID); ok {
			if !timer.timer.Stop() {
				logEntry.Warnf("connection has been timed out, re requesting: %v", connID)
			}
			close(timer.stopCh)
			t.connections.Delete(connID)
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

		t.connections.Store(connID, timer)
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
	timerCtx := extend.WithValuesFromContext(context.Background(), t.ctx)

	timer := &timer{
		stopCh: make(chan struct{}),
	}
	timer.timer = time.AfterFunc(time.Until(expireTime), func() {
		executor, ok := t.executors.Load(conn.GetId())
		if !ok {
			return
		}

		executor.AsyncExec(func() {
			select {
			case <-timer.stopCh:
				logEntry.Warnf("timer has been already stopped: %v", conn.GetId())
			default:
				if err := t.close(timerCtx, conn, next.Server(ctx)); err != nil {
					logEntry.Errorf("failed to close timed out connection: %v %+v", conn.GetId(), err)
				}
			}
		})
	})

	return timer, nil
}

func (t *timeoutServer) Close(ctx context.Context, conn *networkservice.Connection) (_ *empty.Empty, err error) {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "Close")

	executor, ok := t.executors.Load(conn.GetId())
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", conn.GetId())
		return &empty.Empty{}, nil
	}

	<-executor.AsyncExec(func() {
		err = t.close(ctx, conn, next.Server(ctx))
	})

	return &empty.Empty{}, err
}

func (t *timeoutServer) close(ctx context.Context, conn *networkservice.Connection, nextServer networkservice.NetworkServiceServer) error {
	logEntry := log.Entry(ctx).WithField("timeoutServer", "close")

	timer, ok := t.connections.Load(conn.GetId())
	if !ok {
		logEntry.Warnf("connection has been already closed: %v", conn.GetId())
		return nil
	}

	timer.timer.Stop()
	close(timer.stopCh)
	t.connections.Delete(conn.GetId())

	_, err := nextServer.Close(ctx, conn)

	t.executors.Delete(conn.GetId())

	return err
}
