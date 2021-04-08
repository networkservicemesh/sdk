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
	ch         chan *networkservice.PathSegment
	prefixConn *networkservice.Connection
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
		go u.listenToConnectionChanges(pushFunc, errCh)
		u.initErr = <-errCh
	})
	// if initialization failed, then we want for all subsequent calls to Request() on this object to also fail
	if u.initErr != nil {
		return nil, u.initErr
	}

	// TODO what if this is a second call to Request() with this conn id?
	//  will we receive a new event with this channel?
	//  If not, then we will block on waiting and return an error eventually
	verifySuccessCh, err := u.addToExpectedConnections(request.GetConnection())
	if err != nil {
		return nil, err
	}

	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return conn, err
	}

	// Make sure we have a valid expireTime to work with
	//pathSegment := request.GetConnection().GetNextPathSegment()
	//expireTime, err := ptypes.Timestamp(pathSegment.GetExpires())
	//if err != nil {
	//	return conn, errors.Wrapf(err, "error converting pathSegment.GetExpires() to time.Time: %+v", pathSegment.GetExpires())
	//}

successVerifyLoop:
	for {
		select {
		case <-u.ctx.Done():
			u.removeFromExpectedConnections(conn)
			return conn, errors.Errorf("can't wait for connection %v (segment %v) verification: chain context was canceled", conn.GetId(), conn.GetNextPathSegment())
		case segment := <-verifySuccessCh:
			if conn.GetNextPathSegment().Equal(segment) {
				u.removeFromExpectedConnections(conn)
				u.addConnectionToMonitor(conn)
				break successVerifyLoop
			}

			// TODO
			//  original code used expire time for waiting but it didn't matter because
			//  originally we created a monitor and received INITIAL_STATE_TRANSFER event which always was instantaneous,
			//  regardless if there were errors
			//  now it's unclear how long we should wait
		//case <-time.After(time.Until(expireTime)):
		case <-time.After(time.Second):
			u.removeFromExpectedConnections(conn)
			return conn, errors.New("timeout expired but we couldn't verify that connection was established")
		}
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

func (u *healClient) listenToConnectionChanges(heal HealRequestFunc, errCh chan error) {
	monitorClient, err := u.cc.MonitorConnections(u.ctx, &networkservice.MonitorScopeSelector{})
	if err != nil {
		errCh <- errors.Wrap(err, "MonitorConnections failed")
		return
	}
	close(errCh)

	// TODO should we really compare paths from the beginning?
	pathStartsWith := func(c *networkservice.Connection, reference *networkservice.Connection) bool {
		if uint32(len(c.GetPath().GetPathSegments())) <= reference.GetPath().GetIndex() {
			return false
		}
		for i := uint32(0); i < reference.GetPath().GetIndex(); i++ {
			if !c.GetPath().PathSegments[i].Equal(reference.GetPath().GetPathSegments()[i]) {
				return false
			}
		}
		return true
	}

	for u.ctx.Err() == nil {
		event, err := monitorClient.Recv()
		if err != nil {
			if u.ctx.Err() != nil {
				return
			}
			monitorClient, err = u.cc.MonitorConnections(u.ctx, &networkservice.MonitorScopeSelector{})
			if err != nil {
				// TODO can we handle this?
				return
			}
			continue
		}

		var connsToHeal []string

		switch event.GetType() {
		case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER: // TODO is this case special?
			fallthrough
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
			for _, conn := range event.GetConnections() {
				for _, svReq := range u.expectedConns {
					if pathStartsWith(conn, svReq.prefixConn) {
						svReq.ch <- conn.GetPath().GetPathSegments()[svReq.prefixConn.GetPath().GetIndex()+1]
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
			heal(connsToHeal)
		}
	}
}

func (u *healClient) addToExpectedConnections(conn *networkservice.Connection) (chan *networkservice.PathSegment, error) {
	u.expectedConnsMutex.Lock()
	defer u.expectedConnsMutex.Unlock()

	ch := make(chan *networkservice.PathSegment)
	u.expectedConns[conn.GetId()] = successVerifyRequest{
		ch:         ch,
		prefixConn: conn.Clone(),
	}

	return ch, nil
}

func (u *healClient) removeFromExpectedConnections(conn *networkservice.Connection) {
	u.expectedConnsMutex.Lock()
	defer u.expectedConnsMutex.Unlock()

	// TODO this is not a reliable code. Monitor thread could be waiting to write into channel in expectedConns
	delete(u.expectedConns, conn.GetId())
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
