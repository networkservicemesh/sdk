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
	"runtime"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/cancelctx"
)

const maxMonitorAttempts = 10

type connectionInfo struct {
	cond   *sync.Cond
	conn   *networkservice.Connection
	active bool
}

type healClient struct {
	ctx      context.Context
	cc       networkservice.MonitorConnectionClient
	initOnce sync.Once
	initErr  error
	conns    connectionInfoMap
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that monitors its connections' state
//             and calls heal server in case connection breaks if heal server is present in the chain
//             - ctx - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - cc  - MonitorConnectionClient that will be used to watch for connection confirmations and breakages.
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx: ctx,
		cc:  cc,
	}
}

func (u *healClient) init(ctx context.Context, conn *networkservice.Connection) error {
	currentName := conn.GetCurrentPathSegment().GetName()
	createMonitorStream := func() (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
		return u.cc.MonitorConnections(u.ctx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{{Name: currentName}, {Name: ""}},
		}, grpc.WaitForReady(false))
	}

	monitorStream, err := createMonitorStream()
	if err != nil {
		return errors.Wrap(err, "MonitorConnections failed")
	}

	cancel := cancelctx.FromContext(u.ctx)
	if cancel == nil {
		cancel = func() {}
	}

	healConnection := requestHealConnectionFunc(ctx)
	if healConnection == nil {
		healConnection = func(conn *networkservice.Connection) {}
	}
	restoreConnection := requestRestoreConnectionFunc(ctx)
	if restoreConnection == nil {
		restoreConnection = func(conn *networkservice.Connection) {}
	}

	go u.listenToConnectionChanges(monitorStream, createMonitorStream, cancel, healConnection, restoreConnection)

	return nil
}

func (u *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (conn *networkservice.Connection, err error) {
	conn = request.GetConnection()
	u.initOnce.Do(func() {
		u.initErr = u.init(ctx, conn)
	})
	// if initialization failed, then we want for all subsequent calls to Request() on this object to also fail
	if u.initErr != nil {
		return nil, u.initErr
	}

	connInfo, loaded := u.conns.LoadOrStore(conn.GetId(), &connectionInfo{
		conn: conn.Clone(),
		cond: sync.NewCond(new(sync.Mutex)),
	})
	u.replaceConnectionPath(conn, connInfo)

	defer func() {
		if !loaded && err != nil {
			u.conns.Delete(request.GetConnection().GetId())
		}
	}()

	conn, err = next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	condCh, cancel := u.condCh(connInfo)
	defer cancel()

	// Problem:
	//   gRPC requests and streaming requests have different requirements for the underlying gRPC connection. There can
	//   be a case on the closing gRPC connection, when it is still sending requests but fails to keep or start
	//   a streaming request. In such case we can get a situation when we have a Connection with no monitor stream, so
	//   it would never be healed.
	// Solution:
	//   We should check that monitor stream is ready for the Connection, before returning it back. This check should
	//   wait for some time after the next.Request(), because for the [NSMgr -> Endpoint] case response may come faster
	//   than monitor stream update.
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-condCh:
	case <-u.ctx.Done():
		_, _ = next.Client(ctx).Close(ctx, conn)
		return nil, errors.Errorf("timeout waiting for connection monitor: %s", conn.GetId())
	}

	connInfo.cond.L.Lock()
	defer connInfo.cond.L.Unlock()

	connInfo.conn = conn.Clone()

	return conn, nil
}

func (u *healClient) condCh(connInfo *connectionInfo) (ch chan struct{}, cancel func()) {
	ch, condFlag := make(chan struct{}), false
	go func() {
		defer close(ch)

		connInfo.cond.L.Lock()
		defer connInfo.cond.L.Unlock()

		if !condFlag && !connInfo.active {
			connInfo.cond.Wait()
		}
	}()

	return ch, func() {
		connInfo.cond.L.Lock()
		defer connInfo.cond.L.Unlock()

		condFlag = true
		connInfo.cond.Broadcast()
	}
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
func (u *healClient) listenToConnectionChanges(
	monitorStream networkservice.MonitorConnection_MonitorConnectionsClient,
	createMonitorStream func() (networkservice.MonitorConnection_MonitorConnectionsClient, error),
	cancel context.CancelFunc,
	healConnection, restoreConnection requestHealFuncType,
) {
	for attempts := 0; attempts < maxMonitorAttempts && u.ctx.Err() == nil; {
		event, err := monitorStream.Recv()
		if err != nil {
			for attempts++; attempts < maxMonitorAttempts && u.ctx.Err() == nil; {
				runtime.Gosched()
				if monitorStream, err = createMonitorStream(); err != nil {
					attempts++
					continue
				}
			}
			continue
		}
		attempts = 0

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
				connInfo.active = true
				connInfo.conn.Path.PathSegments = eventConn.Clone().Path.PathSegments
				connInfo.conn.NetworkServiceEndpointName = eventConn.NetworkServiceEndpointName
				connInfo.cond.Broadcast()
			case networkservice.ConnectionEventType_DELETE:
				if connInfo.active {
					healConnection(connInfo.conn)
				}
				connInfo.active = false
			}

			connInfo.cond.L.Unlock()
		}
	}
	if u.ctx.Err() != nil {
		return
	}

	cancel()

	u.conns.Range(func(id string, connInfo *connectionInfo) bool {
		connInfo.cond.L.Lock()
		defer connInfo.cond.L.Unlock()

		if connInfo.active {
			restoreConnection(connInfo.conn)
		}
		return true
	})
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
