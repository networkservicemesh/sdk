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
	"time"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/discover"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type ctxWrapper struct {
	conn    *networkservice.Connection
	request *networkservice.NetworkServiceRequest
	ctx     context.Context
	cancel  func()
}

type healServer struct {
	ctx                    context.Context
	onHeal                 *networkservice.NetworkServiceClient
	contextHealMap         map[string]*ctxWrapper
	contextHealMapExecutor serialize.Executor
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
		ctx:            ctx,
		onHeal:         onHeal,
		contextHealMap: make(map[string]*ctxWrapper),
	}

	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(adapters.NewServerToClient(rv))
	}

	return rv
}

func (f *healServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx = withHealRequestFunc(ctx, f.healRequest)
	conn, err := next.Server(ctx).Request(ctx, request)
	if err != nil {
		return conn, err
	}

	ctx = f.applyStoredCandidates(ctx, conn)
	healCtx := f.ctx
	if candidates := discover.Candidates(ctx); candidates != nil {
		healCtx = discover.WithCandidates(healCtx, candidates.Endpoints, candidates.NetworkService)
	}
	<-f.contextHealMapExecutor.AsyncExec(func() {
		id := request.GetConnection().GetId()
		if entry, ok := f.contextHealMap[id]; ok && entry.cancel != nil {
			entry.cancel()
		}
		connCopy := conn.Clone()
		f.contextHealMap[id] = &ctxWrapper{
			conn:    connCopy,
			request: request.Clone().SetRequestConnection(connCopy),
			ctx:     healCtx,
		}
	})

	return conn, nil
}

func (f *healServer) Close(ctx context.Context, conn *networkservice.Connection) (*empty.Empty, error) {
	f.stopHeal(conn)

	rv, err := next.Server(ctx).Close(ctx, conn)
	if err != nil {
		return nil, err
	}
	return rv, nil
}

func (f *healServer) healRequest(conn *networkservice.Connection, restoreConnection bool) {
	var healCtx context.Context
	var request *networkservice.NetworkServiceRequest
	<-f.contextHealMapExecutor.AsyncExec(func() {
		s, ok := f.contextHealMap[conn.GetId()]
		if !ok {
			return
		}
		if s.cancel != nil {
			s.cancel()
		}
		ctx, cancel := context.WithCancel(s.ctx)
		s.cancel = cancel
		healCtx = ctx
		request = s.request
	})

	if request == nil {
		return
	}

	request.SetRequestConnection(conn.Clone())

	if restoreConnection {
		go f.restoreConnection(healCtx, request)
	} else {
		go f.processHeal(healCtx, request)
	}
}

func (f *healServer) stopHeal(conn *networkservice.Connection) {
	<-f.contextHealMapExecutor.AsyncExec(func() {
		if cancelHealEntry, ok := f.contextHealMap[conn.GetId()]; ok {
			if cancelHealEntry.cancel != nil {
				cancelHealEntry.cancel()
			}
			delete(f.contextHealMap, conn.GetId())
		}
	})
}

func (f *healServer) healContext(baseCtx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(f.ctx)

	if candidates := discover.Candidates(baseCtx); candidates != nil {
		ctx = discover.WithCandidates(ctx, candidates.Endpoints, candidates.NetworkService)
	}

	return ctx, cancel
}

func (f *healServer) restoreConnection(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
	if ctx.Err() != nil {
		return
	}

	// Make sure we have a valid expireTime to work with
	expires := request.GetConnection().GetNextPathSegment().GetExpires()
	expireTime, err := ptypes.Timestamp(expires)
	if err != nil {
		return
	}

	deadline := time.Now().Add(time.Minute)
	if deadline.After(expireTime) {
		deadline = expireTime
	}
	requestCtx, requestCancel := context.WithDeadline(ctx, deadline)
	defer requestCancel()

	for requestCtx.Err() == nil {
		if _, err = (*f.onHeal).Request(requestCtx, request.Clone(), opts...); err == nil {
			return
		}
	}

	f.processHeal(ctx, request.Clone(), opts...)
}

func (f *healServer) processHeal(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) {
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
		closeCtx, closeCancel := context.WithTimeout(ctx, time.Second)
		defer closeCancel()

		_, err := (*f.onHeal).Close(closeCtx, request.GetConnection().Clone(), opts...)
		if err != nil {
			logEntry.Errorf("Failed to close connection %s: %v", request.GetConnection().GetId(), err)
		}
	}
}

func (f *healServer) applyStoredCandidates(ctx context.Context, conn *networkservice.Connection) context.Context {
	if candidates := discover.Candidates(ctx); candidates != nil && len(candidates.Endpoints) > 0 {
		return ctx
	}

	var storedCtx context.Context
	<-f.contextHealMapExecutor.AsyncExec(func() {
		if cw, ok := f.contextHealMap[conn.GetId()]; ok {
			storedCtx = cw.ctx
		}
	})
	if storedCtx != nil {
		if candidates := discover.Candidates(storedCtx); candidates != nil {
			ctx = discover.WithCandidates(ctx, candidates.Endpoints, candidates.NetworkService)
		}
	}
	return ctx
}
