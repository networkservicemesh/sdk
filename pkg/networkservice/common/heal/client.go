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

type healClient struct {
	ctx      context.Context
	cc       networkservice.MonitorConnectionClient
	initOnce sync.Once
	initErr  error
	conns    connectionInfoMap
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that inform healServer about new client connection
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx: ctx,
		cc:  cc,
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

	var successVerificationCh chan struct{}
	u.conns.applyLockedOrNew(request.GetConnection().GetId(), func(created bool, info *connectionInfo) {
		conn := request.GetConnection()
		if created {
			successVerificationCh = make(chan struct{}, 1)

			info.conn = conn.Clone()
			info.state = connectionstateAwaitingConfirmation
			info.successVerificationCh = successVerificationCh
		} else if conn.Path != nil && int(conn.Path.Index) < len(conn.Path.PathSegments)-1 {
			storedConn := info.conn
			path := request.GetConnection().Path
			path.PathSegments = path.PathSegments[:path.Index+1]
			path.PathSegments = append(path.PathSegments, storedConn.Path.PathSegments[path.Index+1:]...)
			conn.NetworkServiceEndpointName = storedConn.NetworkServiceEndpointName
		}
	})

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
	u.conns.applyLocked(conn.GetId(), func(info *connectionInfo) {
		info.conn = conn.Clone()
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
			u.conns.Range(func(id string, info *connectionInfo) bool {
				info.mut.Lock()
				defer info.mut.Unlock()
				heal(info.conn, true)
				return true
			})
			return
		}

		for _, eventConn := range event.GetConnections() {
			id := eventConn.GetPrevPathSegment().GetId()
			u.conns.applyLocked(id, func(info *connectionInfo) {
				switch event.GetType() {
				// Why both INITIAL_STATE_TRANSFER and UPDATE:
				// Sometimes we start polling events too late, and when we wait for confirmation of success of some connection,
				// this connection is in the INITIAL_STATE_TRANSFER event, so we must treat these events the same as UPDATE.
				case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
					if info.state == connectionstateAwaitingConfirmation {
						info.successVerificationCh <- struct{}{}
					}
					info.state = connectionstateReady
					info.conn.Path.PathSegments = eventConn.Clone().Path.PathSegments
				case networkservice.ConnectionEventType_DELETE:
					if info.state == connectionstateReady {
						heal(info.conn, false)
					}
					info.state = connectionstateBroken
				}
			})
		}
	}
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
	info, loaded := u.conns.LoadAndDelete(conn.GetId())
	if !loaded {
		return nil, false
	}
	info.mut.Lock()
	defer info.mut.Unlock()
	return info.conn, true
}

func (m *connectionInfoMap) applyLocked(id string, fun func(info *connectionInfo)) {
	info, ok := m.Load(id)
	if !ok {
		return
	}
	info.mut.Lock()
	defer info.mut.Unlock()
	fun(info)
}

func (m *connectionInfoMap) applyLockedOrNew(id string, fun func(created bool, info *connectionInfo)) {
	newInfo := &connectionInfo{}
	// we need to lock this before storing in map to prevent potential race with other threads that can read this map
	newInfo.mut.Lock()
	defer newInfo.mut.Unlock()
	info, loaded := m.LoadOrStore(id, newInfo)
	if loaded {
		info.mut.Lock()
		defer info.mut.Unlock()
	}
	fun(!loaded, info)
}
