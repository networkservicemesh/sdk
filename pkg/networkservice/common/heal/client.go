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

// Package heal provides a chain element that carries out proper nsm healing from client to endpoint
package heal

import (
	"context"
	"sync"
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type connectionState int

const (
	connectionstateAwaitingConfirmation connectionState = iota
	connectionstateReady
	connectionstateBroken
)

type connectionInfo struct {
	conn                  *networkservice.Connection
	state                 connectionState
	successVerificationCh chan struct{}
}

type healClient struct {
	ctx           context.Context
	cc            networkservice.MonitorConnectionClient
	initOnce      sync.Once
	initErr       error
	conns         map[string]connectionInfo
	connsExecutor serialize.Executor
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that inform healServer about new client connection
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx:   ctx,
		cc:    cc,
		conns: make(map[string]connectionInfo),
	}
}

func (u *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	u.initOnce.Do(func() {
		errCh := make(chan error, 1)
		pushFunc := healRequestFunc(ctx)
		if pushFunc == nil {
			pushFunc = func(conn *networkservice.Connection, restoreConnection bool) {}
		}
		go u.listenToConnectionChanges(pushFunc, request.GetConnection().GetCurrentPathSegment(), errCh)
		u.initErr = <-errCh
	})
	// if initialization failed, then we want for all subsequent calls to Request() on this object to also fail
	if u.initErr != nil {
		return nil, u.initErr
	}

	successVerificationCh := u.addToExpectedConnections(request.GetConnection())

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if successVerificationCh != nil {
		err = u.confirmConnectionSuccess(conn, successVerificationCh)
		if err != nil {
			u.removeConnectionFromMonitor(conn)
			_, _ = next.Client(ctx).Close(ctx, request.GetConnection(), opts...)
			return nil, err
		}
	}
	<-u.connsExecutor.AsyncExec(func() {
		connInfo := u.conns[conn.GetId()]
		connInfo.conn = conn.Clone()
		u.conns[conn.GetId()] = connInfo
	})

	return conn, nil
}

func (u *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cachedConn, ok := u.removeConnectionFromMonitor(conn); ok {
		conn = cachedConn
	} else {
		return &empty.Empty{}, nil
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (u *healClient) listenToConnectionChanges(heal healRequestFuncType, currentPathSegment *networkservice.PathSegment, errCh chan error) {
	monitorClient, err := u.cc.MonitorConnections(u.ctx, &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{{Name: currentPathSegment.Name}, {Name: ""}},
	})
	if err != nil {
		errCh <- errors.Wrap(err, "MonitorConnections failed")
		return
	}

	close(errCh)

	for {
		event, err := monitorClient.Recv()
		if err != nil {
			<-u.connsExecutor.AsyncExec(func() {
				for _, connInfo := range u.conns {
					heal(connInfo.conn, true)
				}
			})
			return
		}

		switch event.GetType() {
		// Why both INITIAL_STATE_TRANSFER and UPDATE:
		// 1. Sometimes we start polling events too late, and when we wait for confirmation of success of some connection,
		//    this connection is in the INITIAL_STATE_TRANSFER event, so we must treat these events the same as UPDATE.
		// 2. We can't just try to skip first event right after monitor client creation,
		//    because sometimes we don't get INITIAL_STATE_TRANSFER and therefore we hang indefinitely on Recv().
		case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
			<-u.connsExecutor.AsyncExec(func() {
				for _, eventConn := range event.GetConnections() {
					id := eventConn.GetPrevPathSegment().GetId()
					if connInfo, ok := u.conns[id]; ok {
						if connInfo.state == connectionstateAwaitingConfirmation {
							connInfo.successVerificationCh <- struct{}{}
						}
						connInfo.state = connectionstateReady
						connInfo.conn.Path.PathSegments = eventConn.Clone().Path.PathSegments
						u.conns[id] = connInfo
					}
				}
			})
		case networkservice.ConnectionEventType_DELETE:
			<-u.connsExecutor.AsyncExec(func() {
				for _, eventConn := range event.GetConnections() {
					id := eventConn.GetPrevPathSegment().GetId()
					if connInfo, ok := u.conns[id]; ok {
						if connInfo.state == connectionstateReady {
							heal(connInfo.conn, false)
						}
						connInfo.state = connectionstateBroken
						u.conns[id] = connInfo
					}
				}
			})
		}
	}
}

func (u *healClient) addToExpectedConnections(conn *networkservice.Connection) chan struct{} {
	var successVerificationCh chan struct{}
	<-u.connsExecutor.AsyncExec(func() {
		if _, ok := u.conns[conn.GetId()]; !ok {
			successVerificationCh = make(chan struct{}, 1)
			u.conns[conn.GetId()] = connectionInfo{
				conn:                  conn.Clone(),
				state:                 connectionstateAwaitingConfirmation,
				successVerificationCh: successVerificationCh,
			}
		}
	})

	return successVerificationCh
}

func (u *healClient) confirmConnectionSuccess(conn *networkservice.Connection, successVerificationCh chan struct{}) error {
	timeoutCh := time.After(time.Millisecond * 100)
	select {
	case <-successVerificationCh:
		return nil
	case <-timeoutCh:
		return errors.Errorf("healClient: timeout expired but we couldn't verify that connection was established, connection id: %v", conn.GetId())
	}
}

func (u *healClient) removeConnectionFromMonitor(conn *networkservice.Connection) (*networkservice.Connection, bool) {
	var cachedConn *networkservice.Connection
	var ok bool
	<-u.connsExecutor.AsyncExec(func() {
		var entry connectionInfo
		entry, ok = u.conns[conn.GetId()]
		if ok {
			cachedConn = entry.conn
			delete(u.conns, conn.GetId())
		}
	})

	return cachedConn, ok
}
