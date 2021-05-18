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

// Package onidle provides server chain element that executes a callback when there were no active connections for specified time
package onidle

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/null"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type onIdleServer struct {
	ctx         context.Context
	timeout     time.Duration
	notify      func()
	timer       clock.Timer
	timerMut    sync.Mutex
	timerFired  bool
	activeConns map[string]struct{}
}

// NewServer returns a new server chain element that notifies about long time periods without active connections.
//
// If timeout passes, server calls specified notify function and all further Requests will fail.
//
// Zero timeout disables onidle server altogether. Setting timeout=0 is equivalent to using null.NewServer.
//
// If ctx is canceled before timeout, the server stops monitoring connections without calling notify.
// Further calls to Request will not be affected by this.
func NewServer(ctx context.Context, notify func(), timeout time.Duration) networkservice.NetworkServiceServer {
	if timeout == 0 {
		return null.NewServer()
	}

	t := &onIdleServer{
		ctx:         ctx,
		timeout:     timeout,
		notify:      notify,
		activeConns: make(map[string]struct{}),
	}

	t.timer = clock.FromContext(ctx).AfterFunc(t.timeout, func() {
		if ctx.Err() != nil {
			return
		}

		t.timerMut.Lock()

		if t.timerFired || len(t.activeConns) != 0 {
			t.timerMut.Unlock()
			return
		}

		t.timerFired = true
		t.timerMut.Unlock()
		t.notify()
	})

	go func() {
		<-t.ctx.Done()
		t.timer.Stop()
	}()

	return t
}

func (t *onIdleServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	isRefresh, expired := t.addConnection(request.GetConnection())

	if expired {
		return nil, errors.New("endpoint expired")
	}

	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil && !isRefresh {
		t.removeConnection(request.GetConnection())
	}

	return conn, err
}

func (t *onIdleServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	t.removeConnection(conn)
	return next.Server(ctx).Close(ctx, conn)
}

func (t *onIdleServer) addConnection(conn *networkservice.Connection) (isRefresh, expired bool) {
	t.timerMut.Lock()
	defer t.timerMut.Unlock()

	if t.timerFired {
		return false, true
	}

	if _, isRefresh = t.activeConns[conn.GetId()]; !isRefresh {
		t.activeConns[conn.GetId()] = struct{}{}
		t.timer.Stop()
	}
	return isRefresh, false
}

func (t *onIdleServer) removeConnection(conn *networkservice.Connection) {
	t.timerMut.Lock()
	defer t.timerMut.Unlock()

	if _, loaded := t.activeConns[conn.GetId()]; loaded {
		delete(t.activeConns, conn.GetId())
		if len(t.activeConns) == 0 && t.ctx.Err() == nil {
			t.timer.Reset(t.timeout)
		}
	}
}
