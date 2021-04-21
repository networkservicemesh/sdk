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

type connState int

const (
	connStateInitial connState = iota
	connStateReady
	connStateBroken
)

type connectionInfo struct {
	cond  *sync.Cond
	conn  *networkservice.Connection
	state connState
}

type healClient struct {
	ctx      context.Context
	cc       networkservice.MonitorConnectionClient
	initOnce sync.Once
	initErr  error
	conns    connectionInfoMap
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element
//             - ctx - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - cc  - MonitorConnectionClient that will be used to watch for connection confirmations and breakages.
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx: ctx,
		cc:  cc,
	}
}

func (u *healClient) init(ctx context.Context, conn *networkservice.Connection) error {
	monitorClient, err := u.cc.MonitorConnections(u.ctx, &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{{Name: conn.GetCurrentPathSegment().Name}, {Name: ""}},
	})
	if err != nil {
		return errors.Wrap(err, "MonitorConnections failed")
	}

	healConnection := requestHealConnectionFunc(ctx)
	if healConnection == nil {
		healConnection = func(conn *networkservice.Connection) {}
	}
	restoreConnection := requestRestoreConnectionFunc(ctx)
	if restoreConnection == nil {
		restoreConnection = func(conn *networkservice.Connection) {}
	}

	go u.listenToConnectionChanges(healConnection, restoreConnection, monitorClient)

	return nil
}

func (u *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn := request.GetConnection()
	u.initOnce.Do(func() {
		u.initErr = u.init(ctx, conn)
	})
	// if initialization failed, then we want for all subsequent calls to Request() on this object to also fail
	if u.initErr != nil {
		return nil, u.initErr
	}

	connInfo, _ := u.conns.LoadOrStore(conn.GetId(), &connectionInfo{
		conn: conn.Clone(),
		cond: sync.NewCond(&sync.Mutex{}),
	})
	u.replaceConnectionPath(conn, connInfo)

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	connInfo.cond.L.Lock()
	defer connInfo.cond.L.Unlock()
	connInfo.conn = conn.Clone()
	if connInfo.state != connStateReady {
		ch := make(chan struct{})
		go func() {
			connInfo.cond.Wait()
			close(ch)
		}()
		select {
		case <-ch:
		case <-time.After(100 * time.Millisecond):
			connInfo.cond.Broadcast()
			<-ch
			if connInfo.state == connStateInitial {
				u.conns.Delete(conn.GetId())
			}
			_, _ = next.Client(ctx).Close(ctx, conn, opts...)
			return nil, errors.Errorf("couldn't verify that connection: %v was established", conn.GetId())
		}
	}

	return conn, nil
}

func (u *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	connInfo, loaded := u.conns.LoadAndDelete(conn.GetId())
	if !loaded {
		return &empty.Empty{}, nil
	}
	u.replaceConnectionPath(conn, connInfo)

	return next.Client(ctx).Close(ctx, conn, opts...)
}

// listenToConnectionChanges - loops on events from MonitorConnectionClient while the monitor client is alive.
//                             Updates connection cache and broadcasts events of successful connections.
//                             Calls heal when something breaks.
func (u *healClient) listenToConnectionChanges(healConnection requestHealFuncType, restoreConnection requestHealFuncType, monitorClient networkservice.MonitorConnection_MonitorConnectionsClient) {
	for {
		event, err := monitorClient.Recv()
		if err != nil {
			u.conns.Range(func(id string, info *connectionInfo) bool {
				info.cond.L.Lock()
				defer info.cond.L.Unlock()
				restoreConnection(info.conn)
				return true
			})
			return
		}

		for _, eventConn := range event.GetConnections() {
			connID := eventConn.GetPrevPathSegment().GetId()
			connInfo, ok := u.conns.Load(connID)
			if !ok {
				continue
			}
			connInfo.cond.L.Lock()
			switch event.GetType() {
			// Why both INITIAL_STATE_TRANSFER and UPDATE:
			// Sometimes we start polling events too late, and when we wait for confirmation of success of some connection,
			// this connection is in the INITIAL_STATE_TRANSFER event, so we must treat these events the same as UPDATE.
			case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
				connInfo.state = connStateReady
				connInfo.conn.Path.PathSegments = eventConn.Clone().Path.PathSegments
				connInfo.conn.NetworkServiceEndpointName = eventConn.NetworkServiceEndpointName
				connInfo.cond.Broadcast()
			case networkservice.ConnectionEventType_DELETE:
				if connInfo.state == connStateReady {
					healConnection(connInfo.conn)
				}
				connInfo.state = connStateBroken
			}
			connInfo.cond.L.Unlock()
		}
	}
}

func (u *healClient) replaceConnectionPath(conn *networkservice.Connection, connInfo *connectionInfo) {
	path := conn.GetPath()
	if path != nil && int(path.Index) < len(path.PathSegments)-1 {
		connInfo.cond.L.Lock()
		defer connInfo.cond.L.Unlock()
		path.PathSegments = path.PathSegments[:path.Index+1]
		path.PathSegments = append(path.PathSegments, connInfo.conn.Path.PathSegments[path.Index+1:]...)
		conn.NetworkServiceEndpointName = connInfo.conn.NetworkServiceEndpointName
	}
}
