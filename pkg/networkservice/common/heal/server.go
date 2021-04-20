// Copyright (c) 2020 Cisco Systems, Inc.
//
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
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ctxWrapper struct {
	ctx    context.Context
	cancel func()
}

type clientConnInfo struct {
	ctx context.Context
	cc  networkservice.MonitorConnectionClient
}

type connection struct {
	*networkservice.Connection
	ctx context.Context
}

type healServer struct {
	ctx                   context.Context
	clients               clientConnMap
	onHeal                *networkservice.NetworkServiceClient
	cancelHealMap         map[string]*ctxWrapper
	cancelHealMapExecutor serialize.Executor
	conns                 connectionMap
}

// NewServer - creates a new networkservice.NetworkServiceServer chain element that implements the healing algorithm
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
//             - client - networkservice.MonitorConnectionClient that can be used to call MonitorConnection against the endpoint
//             - onHeal - *networkservice.NetworkServiceClient.  Since networkservice.NetworkServiceClient is an interface
//                        (and thus a pointer) *networkservice.NetworkServiceClient is a double pointer.  Meaning it
//                        points to a place that points to a place that implements networkservice.NetworkServiceClient
//                        This is done because when we use heal.NewClient as part of a chain, we may not *have*
//                        a pointer to this
//                        client used 'onHeal'.  If we detect we need to heal, onHeal.Request is used to heal.
//                        If onHeal is nil, then we simply set onHeal to this client chain element
//                        If we are part of a larger chain or a server, we should pass the resulting chain into
//                        this constructor before we actually have a pointer to it.
//                        If onHeal nil, onHeal will be pointed to the returned networkservice.NetworkServiceClient
func NewServer(ctx context.Context, onHeal *networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	rv := &healServer{
		ctx:           ctx,
		onHeal:        onHeal,
		cancelHealMap: make(map[string]*ctxWrapper),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(adapters.NewServerToClient(rv))
	}

	return rv
}

func (f *healServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	// Replace path within connection to the healed one
	f.replaceConnectionPath(request.GetConnection())

	ctx = withRegisterClientFunc(ctx, f.RegisterClient)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return nil, err
	}

	// Check heal client is in the chain and connection was added
	if _, ok := f.clients.Load(conn.GetId()); !ok {
		log.FromContext(ctx).WithField("healServer", "Request").Errorf("Heal server ignored connection %s: heal client is not active", conn.GetId())
		return conn, nil
	}

	ctx = f.applyStoredCandidates(ctx, conn)

	err = f.startHeal(ctx, request.Clone().SetRequestConnection(conn.Clone()))
	if err != nil {
		return nil, err
	}

	f.conns.Store(conn.GetId(), connection{
		Connection: conn.Clone(),
		ctx:        ctx,
	})

	return conn, nil
}

func (f *healServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	// Replace path within connection to the healed one
	f.replaceConnectionPath(conn)

	f.stopHeal(conn)
	f.clients.Delete(conn.Id)

	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func (f *healServer) RegisterClient(ctx context.Context, conn *networkservice.Connection, cc networkservice.MonitorConnectionClient) {
	f.clients.Store(conn.Id, clientConnInfo{
		ctx: ctx,
		cc:  cc,
	})
}

func (f *healServer) stopHeal(conn *networkservice.Connection) {
	var cancelHeal func()
	<-f.cancelHealMapExecutor.AsyncExec(func() {
		if cancelHealEntry, ok := f.cancelHealMap[conn.GetId()]; ok {
			cancelHeal = cancelHealEntry.cancel
			delete(f.cancelHealMap, conn.GetId())
		}
	})
	if cancelHeal != nil {
		cancelHeal()
	}
	f.conns.Delete(conn.GetId())
}

// startHeal - start a healAsNeeded using the request as the request for re-request if healing is needed.
func (f *healServer) startHeal(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) error {
	errCh := make(chan error, 1)
	go f.healAsNeeded(ctx, request, errCh, opts...)
	return <-errCh
}

// healAsNeeded - heal the connection found in request.  Will immediately do a monitor to make sure the server has the
// expected connection and it is sane, returning an error via errCh if there is an issue, and nil via errCh if there is
// not.  You will only *ever* receive one real error via the errCh.  errCh will be closed when healAsNeeded is finished
// allowing it to double as a 'done' channel we can use when we stopHealing in f.Close.
// healAsNeeded will then continue to monitor the servers opinions about the state of the connection until either
// expireTime has passed or stopHeal is called (as in Close) or a different pathSegment is found via monitoring
// indicating that a later Request has occurred and in doing so created its own healAsNeeded and so we can stop this one
func (f *healServer) healAsNeeded(baseCtx context.Context, request *networkservice.NetworkServiceRequest, errCh chan error, opts ...grpc.CallOption) {
	clockTime := clock.FromContext(baseCtx)

	pathSegment := request.GetConnection().GetNextPathSegment()

	// Make sure we have a valid expireTime to work with
	expireTime, err := ptypes.Timestamp(pathSegment.GetExpires())
	if err != nil {
		errCh <- errors.Wrapf(err, "error converting pathSegment.GetExpires() to time.Time: %+v", pathSegment.GetExpires())
		return
	}

	ctx, cancel := f.healContext(baseCtx)
	defer cancel()
	id := request.GetConnection().GetId()
	<-f.cancelHealMapExecutor.AsyncExec(func() {
		if entry, ok := f.cancelHealMap[id]; ok {
			go entry.cancel() // TODO - what to do with the errCh here?
		}
		f.cancelHealMap[id] = &ctxWrapper{
			ctx:    ctx,
			cancel: cancel,
		}
	})

	// Monitor the pathSegment for the first time, so we can pass back an error
	// if we can't confirm via monitor the other side has the expected state
	recv, err := f.initialMonitorSegment(ctx, request.GetConnection(), clockTime.Until(expireTime))
	if err != nil {
		errCh <- errors.Wrapf(err, "error calling MonitorConnection_MonitorConnectionsClient.Recv to get initial confirmation server has connection: %+v", request.GetConnection())
		return
	}

	// Tell the caller all is well by sending them a nil err so the call can continue
	close(errCh)

	// Start looping over events
	for ctx.Err() == nil {
		event, err := recv.Recv()
		if err != nil {
			deadline := clockTime.Now().Add(time.Minute)
			if deadline.After(expireTime) {
				deadline = expireTime
			}
			newRecv, newRecvErr := f.initialMonitorSegment(ctx, request.GetConnection(), clockTime.Until(deadline))
			if newRecvErr == nil {
				recv = newRecv
			} else {
				f.restoreConnection(ctx, request, opts...)
				return
			}
			runtime.Gosched()
			continue
		}
		if ctx.Err() != nil {
			return
		}
		if err := f.processEvent(ctx, request, event, opts...); err != nil {
			return
		}
	}
}

func (f *healServer) healContext(baseCtx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(f.ctx)

	if candidates := discover.Candidates(baseCtx); candidates != nil {
		ctx = discover.WithCandidates(ctx, candidates.Endpoints, candidates.NetworkService)
	}

	return ctx, cancel
}

// initialMonitorSegment - monitors for pathSegment and returns a recv and an error if the server does not have
// a record for the connection matching our expectations
func (f *healServer) initialMonitorSegment(ctx context.Context, conn *networkservice.Connection, timeout time.Duration) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	clockTime := clock.FromContext(ctx)

	errCh := make(chan error, 1)
	var recv networkservice.MonitorConnection_MonitorConnectionsClient
	pathSegment := conn.GetNextPathSegment()

	// nolint:govet
	initialCtx, cancel := context.WithCancel(ctx)

	go func() {
		// If pathSegment is nil, the server is very very screwed up
		if pathSegment == nil {
			errCh <- errors.New("pathSegment for server connection must not be nil")
			return
		}

		// Get connection client by connection
		client, ok := f.clients.Load(conn.GetId())
		if !ok {
			errCh <- errors.Errorf("error when attempting to MonitorConnections")
			return
		}

		// Monitor *just* this connection
		var err error
		recv, err = client.cc.MonitorConnections(initialCtx, &networkservice.MonitorScopeSelector{
			PathSegments: []*networkservice.PathSegment{
				pathSegment,
			},
		})
		if err != nil {
			errCh <- errors.Wrap(err, "error when attempting to MonitorConnections")
			return
		}

		// Get an initial event to make sure we have the expected connection
		event, err := recv.Recv()
		if err != nil {
			errCh <- err
			return
		}
		// If we didn't get an event something very bad has happened
		if event.Connections == nil || event.Connections[pathSegment.GetId()] == nil {
			errCh <- errors.Errorf("connection with id %s not found in MonitorConnections event as expected: event: %+v", pathSegment.Id, event)
			return
		}
		// If its not *our* connection something's gone wrong like a later Request succeeding
		if !pathSegment.Equal(event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment()) {
			errCh <- errors.Errorf("server reports a different connection for this id, pathSegments do not match.  Expected: %+v Received %+v", pathSegment, event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment())
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		// nolint:govet
		return recv, err
	case <-clockTime.After(timeout):
		cancel()
		err := <-errCh
		return recv, err
	}
}

// processEvent - process event, calling (*f.OnHeal).Request(ctx,request,opts...) if the server does not have our connection.
// returns a non-nil error if the event is such that we should no longer to continue to attempt to heal.
func (f *healServer) processEvent(ctx context.Context, request *networkservice.NetworkServiceRequest, event *networkservice.ConnectionEvent, opts ...grpc.CallOption) error {
	pathSegment := request.GetConnection().GetNextPathSegment()

	switch event.GetType() {
	case networkservice.ConnectionEventType_UPDATE:
		// We should never receive an UPDATE that isn't ours, but in case we do...
		if event.Connections == nil || event.Connections[pathSegment.GetId()] == nil {
			break
		}
		fallthrough
	case networkservice.ConnectionEventType_INITIAL_STATE_TRANSFER:
		if event.Connections != nil && event.Connections[pathSegment.GetId()] != nil {
			// If the server has a pathSegment for this Connection.Id, but its not the one we
			// got back from it... we should fail, as different Request came after ours successfully
			if !pathSegment.Equal(event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment()) {
				return errors.Errorf("server has a different pathSegment than was returned to this call.")
			}
			break
		}
		fallthrough
	case networkservice.ConnectionEventType_DELETE:
		if event.Connections != nil && event.Connections[pathSegment.GetId()] != nil {
			f.processHeal(ctx, request, opts...)
		}
	}
	return nil
}

func (f *healServer) restoreConnection(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
	clockTime := clock.FromContext(ctx)

	id := request.GetConnection().GetId()

	var healMapCtx context.Context
	<-f.cancelHealMapExecutor.AsyncExec(func() {
		if entry, ok := f.cancelHealMap[id]; ok {
			healMapCtx = entry.ctx
		}
	})
	if healMapCtx != ctx || ctx.Err() != nil {
		return
	}

	// Make sure we have a valid expireTime to work with
	expires := request.GetConnection().GetNextPathSegment().GetExpires()
	expireTime, err := ptypes.Timestamp(expires)
	if err != nil {
		return
	}

	deadline := clockTime.Now().Add(time.Minute)
	if deadline.After(expireTime) {
		deadline = expireTime
	}
	requestCtx, requestCancel := clockTime.WithDeadline(ctx, deadline)
	defer requestCancel()

	for requestCtx.Err() == nil {
		if _, err = (*f.onHeal).Request(requestCtx, request.Clone(), opts...); err == nil {
			return
		}
	}

	f.processHeal(ctx, request.Clone(), opts...)
}

func (f *healServer) processHeal(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
	clockTime := clock.FromContext(ctx)
	logEntry := log.FromContext(ctx).WithField("healServer", "processHeal")

	conn := request.GetConnection()

	candidates := discover.Candidates(ctx)
	if candidates != nil || conn.GetPath().GetIndex() == 0 {
		logEntry.Infof("Starting heal process for %s", conn.GetId())

		healCtx, healCancel := context.WithCancel(ctx)
		defer healCancel()

		reRequest := request.Clone()
		reRequest.GetConnection().NetworkServiceEndpointName = ""
		path := reRequest.GetConnection().Path
		reRequest.GetConnection().Path.PathSegments = path.PathSegments[0 : path.Index+1]

		for ctx.Err() == nil {
			_, err := (*f.onHeal).Request(healCtx, reRequest, opts...)
			if err != nil {
				logEntry.Errorf("Failed to heal connection %s: %v", conn.GetId(), err)
			} else {
				logEntry.Infof("Finished heal process for %s", conn.GetId())
				break
			}
		}
	} else {
		// Huge timeout is not required to close connection on a current path segment
		closeCtx, closeCancel := clockTime.WithTimeout(ctx, time.Second)
		defer closeCancel()

		_, err := (*f.onHeal).Close(closeCtx, request.GetConnection().Clone(), opts...)
		if err != nil {
			logEntry.Errorf("Failed to close connection %s: %v", request.GetConnection().GetId(), err)
		}
	}
}

func (f *healServer) replaceConnectionPath(conn *networkservice.Connection) {
	path := conn.GetPath()
	if path != nil && int(path.Index) < len(path.PathSegments)-1 {
		if storedConn, ok := f.conns.Load(conn.GetId()); ok {
			path.PathSegments = path.PathSegments[:path.Index+1]
			path.PathSegments = append(path.PathSegments, storedConn.Path.PathSegments[path.Index+1:]...)
			conn.NetworkServiceEndpointName = storedConn.NetworkServiceEndpointName
		}
	}
}

func (f *healServer) applyStoredCandidates(ctx context.Context, conn *networkservice.Connection) context.Context {
	if candidates := discover.Candidates(ctx); candidates != nil && len(candidates.Endpoints) > 0 {
		return ctx
	}

	if info, ok := f.conns.Load(conn.Id); ok {
		if candidates := discover.Candidates(info.ctx); candidates != nil {
			ctx = discover.WithCandidates(ctx, candidates.Endpoints, candidates.NetworkService)
		}
	}
	return ctx
}
