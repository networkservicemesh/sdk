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
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"sync"
	"time"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type successVerifyRequest struct {
	connectionSuccessCh chan *networkservice.PathSegment
	stopCh              chan struct{}
}

type healClient struct {
	ctx      context.Context
	cc       networkservice.MonitorConnectionClient
	initOnce sync.Once
	initErr  error

	// connCache contains connections which IDs we got in Request() updated to actual values as soon as monitor tells us something's changed
	connCache      map[string]*networkservice.Connection
	connCacheMutex sync.Mutex

	// expectedConns contains connections we will be waiting to happen
	expectedConns      map[string]successVerifyRequest
	expectedConnsMutex sync.Mutex
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that inform healServer about new client connection
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx:           ctx,
		cc:            cc,
		connCache:     make(map[string]*networkservice.Connection),
		expectedConns: make(map[string]successVerifyRequest),
	}
}

func (u *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	u.initOnce.Do(func() {
		errCh := make(chan error, 1)
		pushFunc := healRequestFunc(ctx)
		go u.listenToConnectionChanges(pushFunc, request.GetConnection().GetCurrentPathSegment(), errCh)
		u.initErr = <-errCh
	})
	// if initialization failed, then we want for all subsequent calls to Request() on this object to also fail
	if u.initErr != nil {
		return nil, u.initErr
	}

	req, err := u.addToExpectedConnections(request.GetConnection())
	if req != nil {
		defer close(req.stopCh) // TODO try closing earlier?
	}
	if err != nil {
		return nil, err
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if req != nil {
	successVerifyLoop:
		for {
			select {
			case segment := <-req.connectionSuccessCh:
				if conn.GetNextPathSegment().Equal(segment) {
					u.addConnectionToMonitor(conn)
					break successVerifyLoop
				}
			case <-time.After(time.Millisecond * 100):
				return nil, errors.Errorf("healClient: timeout expired but we couldn't verify that connection was established, connection id: %v", conn.GetId())
			}
		}
	} else {
		u.connCacheMutex.Lock()
		u.connCache[conn.GetId()] = conn.Clone()
		u.connCacheMutex.Unlock()
	}

	return conn, nil
}

func (u *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cachedConn, err := u.removeConnectionFromMonitor(conn); err == nil {
		conn = cachedConn
	} else {
		// TODO log this?
	}

	return next.Client(ctx).Close(ctx, conn, opts...)
}

func (u *healClient) listenToConnectionChanges(heal HealRequestFunc, currentPathSegment *networkservice.PathSegment, errCh chan error) {
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
			if heal != nil {
				u.connCacheMutex.Lock()
				for id := range u.connCache {
					heal(id, true)
				}
				u.connCacheMutex.Unlock()
			}
			return
		}

		var connsToHeal []string

		switch event.GetType() {
		case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
			continue
		case networkservice.ConnectionEventType_UPDATE:
			// TODO adapt this code, copied from server, to client
			// If the server has a pathSegment for this Connection.Id, but its not the one we
			// got back from it... we should fail, as different Request came after ours successfully
			//if !pathSegment.Equal(conns[pathSegment.GetId()].GetCurrentPathSegment()) {
			//	return errors.Errorf("server has a different pathSegment than was returned to this call.")
			//}

			// todo fix this mutex mess
			u.connCacheMutex.Lock()
			for _, conn := range event.GetConnections() {
				if _, ok := u.connCache[conn.GetId()]; ok {
					u.connCache[conn.GetId()] = conn.Clone()
				}
			}
			u.connCacheMutex.Unlock()

			u.expectedConnsMutex.Lock()
		tmp_label:
			for _, conn := range event.GetConnections() {
				for reqId, reqChs := range u.expectedConns {
					if reqId == conn.GetPrevPathSegment().GetId() {
						select {
						case <-reqChs.stopCh:
							delete(u.expectedConns, reqId)
							break tmp_label
						case reqChs.connectionSuccessCh <- conn.GetCurrentPathSegment():
							break
						}
					}
				}
			}
			u.expectedConnsMutex.Unlock()

		case networkservice.ConnectionEventType_DELETE:
			if heal != nil {
				u.connCacheMutex.Lock()
				for _, eventConn := range event.GetConnections() {
					if conn, ok := u.connCache[eventConn.GetPrevPathSegment().GetId()]; ok {
						connsToHeal = append(connsToHeal, conn.GetId())
					}
				}
				u.connCacheMutex.Unlock()
			}
		}

		if heal != nil {
			for _, id := range connsToHeal {
				heal(id, false)
			}
		}
	}
}

func (u *healClient) addToExpectedConnections(conn *networkservice.Connection) (*successVerifyRequest, error) {
	connectionWasEstablishedPreviously := false
	u.connCacheMutex.Lock()
	if cachedConn, ok := u.connCache[conn.GetId()]; ok {
		if cachedConn.GetCurrentPathSegment().GetId() == conn.GetCurrentPathSegment().GetId() {
			connectionWasEstablishedPreviously = true
		}
	}
	u.connCacheMutex.Unlock()

	if connectionWasEstablishedPreviously {
		return nil, nil
	}

	req := successVerifyRequest{
		connectionSuccessCh: make(chan *networkservice.PathSegment, 10),
		stopCh:              make(chan struct{}),
	}

	u.expectedConnsMutex.Lock()
	defer u.expectedConnsMutex.Unlock()

	u.expectedConns[conn.GetId()] = req

	return &req, nil
}

func (u *healClient) addConnectionToMonitor(conn *networkservice.Connection) {
	u.connCacheMutex.Lock()
	defer u.connCacheMutex.Unlock()
	u.connCache[conn.GetId()] = conn.Clone()
}

func (u *healClient) removeConnectionFromMonitor(conn *networkservice.Connection) (*networkservice.Connection, error) {
	u.connCacheMutex.Lock()
	defer u.connCacheMutex.Unlock()
	cachedConn, ok := u.connCache[conn.GetId()]
	if !ok {
		return nil, errors.Errorf("can't find connection with id %s in cache", conn.GetId())
	}
	delete(u.connCache, conn.GetId())

	u.expectedConnsMutex.Lock()
	defer u.expectedConnsMutex.Unlock()

	// in case we are closing before we got any confirmation that connection was established
	delete(u.expectedConns, conn.GetId())

	return cachedConn, nil
}
