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
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type healClient struct {
	ctx              context.Context
	cc               networkservice.MonitorConnectionClient
	initOnce         sync.Once
	initErr          error
	connCache        map[string]*networkservice.Connection
	conCacheExecutor serialize.Executor
	expectedConns    verificationRequestMap
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that inform healServer about new client connection
func NewClient(ctx context.Context, cc networkservice.MonitorConnectionClient) networkservice.NetworkServiceClient {
	return &healClient{
		ctx:       ctx,
		cc:        cc,
		connCache: make(map[string]*networkservice.Connection),
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

	verificationRequest := u.addToExpectedConnections(request.GetConnection())
	if verificationRequest != nil {
		defer close(verificationRequest.stopCh)
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}

	if verificationRequest != nil {
		err = u.confirmConnectionSuccess(conn, verificationRequest)
		if err != nil {
			return nil, err
		}
	} else {
		u.addConnectionToMonitor(conn)
	}

	return conn, nil
}

func (u *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	if cachedConn, ok := u.removeConnectionFromMonitor(conn); ok {
		conn = cachedConn
	} else {
		log.FromContext(ctx).WithField("healClient", "Close").Warnf("can't find connection in cache, id=%v", conn.GetId())
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
			if heal != nil {
				<-u.conCacheExecutor.AsyncExec(func() {
					for id := range u.connCache {
						heal(id, true)
					}
				})
			}
			return
		}

		switch event.GetType() {
		// Sometimes we start polling events too late, and when we wait for confirmation of success of some connection,
		// this connection is in the INITIAL_STATE_TRANSFER event, so we must treat these events the same as UPDATE.
		// We can't just try to skip first event right after monitor client creation,
		// because sometimes we don't get INITIAL_STATE_TRANSFER and therefore we hang indefinitely on Recv().
		case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER, networkservice.ConnectionEventType_UPDATE:
			<-u.conCacheExecutor.AsyncExec(func() {
				for _, conn := range event.GetConnections() {
					if _, ok := u.connCache[conn.GetId()]; ok {
						u.connCache[conn.GetId()] = conn.Clone()
					}
				}
			})

			u.checkExpectedConnections(event.GetConnections())

		case networkservice.ConnectionEventType_DELETE:
			if heal != nil {
				var connsToHeal []string
				<-u.conCacheExecutor.AsyncExec(func() {
					for _, eventConn := range event.GetConnections() {
						if conn, ok := u.connCache[eventConn.GetPrevPathSegment().GetId()]; ok {
							connsToHeal = append(connsToHeal, conn.GetId())
						}
					}
				})
				for _, id := range connsToHeal {
					heal(id, false)
				}
			}
		}
	}
}

func (u *healClient) checkExpectedConnections(eventConns map[string]*networkservice.Connection) {
	u.expectedConns.Range(func(reqId string, reqChs connVerificationRequest) bool {
		for _, conn := range eventConns {
			if reqId == conn.GetPrevPathSegment().GetId() {
				select {
				case <-reqChs.stopCh:
					u.expectedConns.Delete(reqId)
					return false
				case reqChs.connectionSuccessCh <- conn.GetCurrentPathSegment():
					break
				}
			}
		}
		return true
	})
}

func (u *healClient) addToExpectedConnections(conn *networkservice.Connection) *connVerificationRequest {
	connAlreadyExists := false
	<-u.conCacheExecutor.AsyncExec(func() {
		if cachedConn, ok := u.connCache[conn.GetId()]; ok {
			if cachedConn.GetCurrentPathSegment().GetId() == conn.GetCurrentPathSegment().GetId() {
				connAlreadyExists = true
			}
		}
	})

	if connAlreadyExists {
		return nil
	}

	req := connVerificationRequest{
		connectionSuccessCh: make(chan *networkservice.PathSegment, 10),
		stopCh:              make(chan struct{}),
	}

	u.expectedConns.Store(conn.GetId(), req)

	return &req
}

func (u *healClient) confirmConnectionSuccess(conn *networkservice.Connection, req *connVerificationRequest) error {
	timeoutCh := time.After(time.Millisecond * 100)
	for {
		select {
		case segment := <-req.connectionSuccessCh:
			if conn.GetNextPathSegment().Equal(segment) {
				u.addConnectionToMonitor(conn)
				return nil
			}
		case <-timeoutCh:
			return errors.Errorf("healClient: timeout expired but we couldn't verify that connection was established, connection id: %v", conn.GetId())
		}
	}
}

func (u *healClient) addConnectionToMonitor(conn *networkservice.Connection) {
	<-u.conCacheExecutor.AsyncExec(func() {
		u.connCache[conn.GetId()] = conn.Clone()
	})
}

func (u *healClient) removeConnectionFromMonitor(conn *networkservice.Connection) (*networkservice.Connection, bool) {
	var cachedConn *networkservice.Connection
	var ok bool
	<-u.conCacheExecutor.AsyncExec(func() {
		cachedConn, ok = u.connCache[conn.GetId()]
		if ok {
			delete(u.connCache, conn.GetId())
		}
	})

	return cachedConn, ok
}
