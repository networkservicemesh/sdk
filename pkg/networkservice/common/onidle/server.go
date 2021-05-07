// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package onidle provides server chain element that executes a callback when there were no connections for specified time
package onidle

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type onIdleServer struct {
	ctx         context.Context
	timeout     time.Duration
	notify      func()
	activeConns sync.Map
	timer       clock.Timer
	// Timers don't support concurrency natively.
	// If we stop it, they tell us if the timer was running before our call.
	// But if timer was not running, there's no way to distinguish
	// if it was stopped earlier (e.g. concurrently by another thread) or if it has already fired.
	// Therefore, we should implement some manual sync for this.
	timerMut         sync.Mutex
	timerStopRequest bool
	timerFired       bool
}

// NewServer - returns a new server chain element that notifies about long time periods without active requests
func NewServer(ctx context.Context, notify func(), options ...Option) networkservice.NetworkServiceServer {
	clockTime := clock.FromContext(ctx)

	t := &onIdleServer{
		ctx:     ctx,
		timeout: time.Minute * 10,
		notify:  notify,
	}

	for _, opt := range options {
		opt(t)
	}

	t.timer = clockTime.Timer(t.timeout)

	go t.waitForTimeout()

	return t
}

func (t *onIdleServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	_, isRefresh := t.activeConns.LoadOrStore(request.GetConnection().GetId(), struct{}{})

	expired := t.stopTimer()
	if expired {
		t.removeConnection(request.GetConnection())
		return nil, errors.New("endpoint expired")
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		if !isRefresh {
			t.removeConnection(request.GetConnection())
		}
	}

	return conn, err
}

func (t *onIdleServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	t.removeConnection(conn)
	return next.Server(ctx).Close(ctx, conn)
}

func (t *onIdleServer) waitForTimeout() {
	for {
		select {
		case <-t.ctx.Done():
			return
		case <-t.timer.C():
			t.timerMut.Lock()
			if !t.timerStopRequest {
				t.timerFired = true
				t.timerMut.Unlock()
				t.notify()
				return
			}
			t.timerMut.Unlock()
		}
	}
}

// stopTimer - stops the timer. Returns true if it has already fired, false otherwise
func (t *onIdleServer) stopTimer() bool {
	t.timerMut.Lock()
	defer t.timerMut.Unlock()

	if t.timerFired {
		return true
	}

	t.timerStopRequest = true
	t.timer.Stop()

	return false
}

func (t *onIdleServer) removeConnection(conn *networkservice.Connection) {
	t.activeConns.Delete(conn.GetId())
	t.startTimerIfNoActiveConns()
}

func (t *onIdleServer) startTimerIfNoActiveConns() {
	any := false
	t.activeConns.Range(func(key, value interface{}) bool {
		any = true
		return false
	})
	if !any {
		t.timerMut.Lock()
		defer t.timerMut.Unlock()

		t.timerStopRequest = false
		t.timer.Reset(t.timeout)
	}
}
