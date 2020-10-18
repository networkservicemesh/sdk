// Copyright (c) 2020 Cisco Systems, Inc.
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

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/tools/addressof"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type healClient struct {
	ctx                   context.Context
	client                networkservice.MonitorConnectionClient
	onHeal                *networkservice.NetworkServiceClient
	cancelHealMap         map[string]func() <-chan error
	cancelHealMapExecutor serialize.Executor
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that implements the healing algorithm
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
func NewClient(ctx context.Context, client networkservice.MonitorConnectionClient, onHeal *networkservice.NetworkServiceClient) networkservice.NetworkServiceClient {
	rv := &healClient{
		ctx:           ctx,
		client:        client,
		onHeal:        onHeal,
		cancelHealMap: make(map[string]func() <-chan error),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(rv)
	}

	return rv
}

func (f *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	conn, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, err
	}
	err = f.startHeal(ctx, request.Clone().SetRequestConnection(conn.Clone()), opts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (f *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	f.stopHeal(conn)
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func (f *healClient) stopHeal(conn *networkservice.Connection) {
	var cancelHeal func() <-chan error
	<-f.cancelHealMapExecutor.AsyncExec(func() {
		cancelHeal = f.cancelHealMap[conn.GetId()]
	})
	<-cancelHeal()
}

// startHeal - start a healAsNeeded using the request as the request for re-request if healing is needed.
func (f *healClient) startHeal(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) error {
	id := request.GetConnection().GetId()

	ctx, cancel := context.WithCancel(ctx)

	errCh := make(chan error, 1)
	f.cancelHealMapExecutor.AsyncExec(func() {
		if cancel, ok := f.cancelHealMap[id]; ok {
			go cancel() // TODO - what to do with the errCh here?
		}
		f.cancelHealMap[id] = func() <-chan error {
			cancel()
			return errCh
		}
	})
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
func (f *healClient) healAsNeeded(ctx context.Context, request *networkservice.NetworkServiceRequest, errCh chan error, opts ...grpc.CallOption) {
	// When we are done, close the errCh
	defer close(errCh)

	pathSegment := request.GetConnection().GetNextPathSegment()

	// Monitor the pathSegment - the first time with the calls context, so we can pass back and error
	// if we can't confirm via monitor the other side has the expected state
	recv, err := f.initialMonitorSegment(ctx, pathSegment)
	if err != nil {
		errCh <- errors.Wrapf(err, "error calling MonitorConnection_MonitorConnectionsClient.Recv to get initial confirmation server has connection: %+v", request.GetConnection())
		return
	}

	// Make sure we have a valid expireTime to work with
	expireTime, err := ptypes.Timestamp(pathSegment.GetExpires())
	if err != nil {
		errCh <- errors.Wrapf(err, "error converting pathSegment.GetExpires() to time.Time: %+v", pathSegment.GetExpires())
		return
	}

	// Set the ctx Deadline to expireTime based on the heal servers context
	ctx, cancel := context.WithDeadline(f.ctx, expireTime)
	defer cancel()
	id := request.GetConnection().GetId()
	f.cancelHealMapExecutor.AsyncExec(func() {
		if cancel, ok := f.cancelHealMap[id]; ok {
			go cancel() // TODO - what to do with the errCh here?
		}
		f.cancelHealMap[id] = func() <-chan error {
			cancel()
			return errCh
		}
	})

	// Tell the caller all is well by sending them a nil err so the call can continue
	errCh <- nil

	// Start looping over events
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		event, err := recv.Recv()
		if err != nil {
			// If we get an error, try to get a new recv ... if that fails, loop around and try again until
			// we succeed or the ctx is canceled or expires
			newRecv, newRecvErr := f.client.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{
				PathSegments: []*networkservice.PathSegment{
					pathSegment,
				},
			})
			if newRecvErr == nil {
				recv = newRecv
			}
			runtime.Gosched()
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
		if err := f.processEvent(ctx, request, event, opts...); err != nil {
			if err != nil {
				return
			}
		}
	}
}

// initialMonitorSegment - monitors for pathSegment and returns a recv and an error if the server does not have
// a record for the connection matching our expectations
func (f *healClient) initialMonitorSegment(ctx context.Context, pathSegment *networkservice.PathSegment) (networkservice.MonitorConnection_MonitorConnectionsClient, error) {
	// If pathSegment is nil, the server is very very screwed up
	if pathSegment == nil {
		return nil, errors.New("pathSegment for server connection must not be nil")
	}

	// Monitor *just* this connection
	recv, err := f.client.MonitorConnections(ctx, &networkservice.MonitorScopeSelector{
		PathSegments: []*networkservice.PathSegment{
			pathSegment,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "error when attempting to MonitorConnections")
	}

	// Get an initial event to make sure we have the expected connection
	event, err := recv.Recv()
	if err != nil {
		return nil, err
	}
	// If we didn't get an event something very bad has happened
	if event.Connections == nil || event.Connections[pathSegment.GetId()] == nil {
		return nil, errors.Errorf("connection with id %s not found in MonitorConnections event as expected: event: %+v", pathSegment.Id, event)
	}
	// If its not *our* connection something's gone wrong like a later Request succeeding
	if !pathSegment.Equal(event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment()) {
		return nil, errors.Errorf("server reports a different connection for this id, pathSegments do not match.  Expected: %+v Received %+v", pathSegment, event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment())
	}
	return recv, nil
}

// processEvent - process event, calling (*f.OnHeal).Request(ctx,request,opts...) if the server does not have our connection.
// returns a non-nil error if the event is such that we should no longer to continue to attempt to heal.
func (f *healClient) processEvent(ctx context.Context, request *networkservice.NetworkServiceRequest, event *networkservice.ConnectionEvent, opts ...grpc.CallOption) error {
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
		if event.Connections != nil && event.Connections[pathSegment.Id] != nil && pathSegment.Equal(event.GetConnections()[pathSegment.GetId()].GetCurrentPathSegment()) {
			_, err := (*f.onHeal).Request(ctx, request, opts...)
			for err != nil {
				// Note: ctx here has deadline set to the expireTime of the pathSegment... so there is a finite stop point
				// to trying to heal.  Additionally, a Close on the connection will trigger a cancel on ctx and
				// wait for errCh to finish *before* calling Close down the line... so we won't accidentally
				// recreate a closed connection.
				runtime.Gosched()
				select {
				case <-ctx.Done():
					return nil
				default:
				}
				_, err := (*f.onHeal).Request(ctx, request, opts...)
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
	return nil
}
