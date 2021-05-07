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
	timerMut     sync.Mutex
	timerFired   bool
	timerCounter int
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
	isRefresh, expired := t.addConnection(request.GetConnection())

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
			t.timer.Stop()
			return
		case <-t.timer.C():
			t.timerMut.Lock()
			if t.timerCounter == 0 {
				t.timerFired = true
				t.timerMut.Unlock()
				t.notify()
				return
			}
			t.timerMut.Unlock()
		}
	}
}

func (t *onIdleServer) addConnection(conn *networkservice.Connection) (isRefresh, expired bool) {
	t.timerMut.Lock()
	defer t.timerMut.Unlock()

	if t.timerFired {
		return false, true
	}

	_, isRefresh = t.activeConns.LoadOrStore(conn.GetId(), struct{}{})
	if !isRefresh {
		t.timerCounter++
		t.timer.Stop()
	}
	return
}

func (t *onIdleServer) removeConnection(conn *networkservice.Connection) {
	t.timerMut.Lock()
	defer t.timerMut.Unlock()

	_, loaded := t.activeConns.LoadAndDelete(conn.GetId())
	if loaded {
		t.timerCounter--
		if t.timerCounter == 0 {
			t.timer.Reset(t.timeout)
		}
	}
}
