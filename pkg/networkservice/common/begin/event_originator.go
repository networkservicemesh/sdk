// Copyright (c) 2021 Cisco and/or its affiliates.
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

package begin

import (
	"context"

	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

// EventOriginator provides a handle for midchain elements to call Request() or Close() against the beginning of
// a chain
type EventOriginator struct {
	// retryRequest *networkservice.NetworkServiceRequest
	executor serialize.Executor
	req      *networkservice.NetworkServiceRequest
	conn     *networkservice.Connection
	opts     []grpc.CallOption

	nextCtx context.Context

	client networkservice.NetworkServiceClient
	server networkservice.NetworkServiceServer
}

func newEventOriginator(nextCtx context.Context, server networkservice.NetworkServiceServer, client networkservice.NetworkServiceClient) *EventOriginator {
	return &EventOriginator{
		nextCtx: nextCtx,
		client:  client,
		server:  server,
	}
}

/*
Request - calls the begin.Request(ctx,request) from the beginning of the chain.

Uses the networkservice.NetworkServiceRequest from the chain's last successfully completed Request()
with networkservice.NetworkServiceRequest.Connection replaced with the Connection returned by the chain's last
successfully completed Request() event

Use:

	func WithRequestMutator(parent context.Context, mutators ...RequestMutatorFunc) context.Context

to mutate the default Request sent.

Note: if a chain is a server chain continued by a client chain, the beginning of the chain is at the beginning of
the server chain, even if there is a subsequent begin.NewClient() in the client chain.
*/
func (e *EventOriginator) Request(ctx context.Context) (conn *networkservice.Connection, err error) {
	if e == nil {
		return nil, errors.Errorf("EventOriginator.Request should not be calle on a nil receiver")
	}
	req := e.req.Clone()
	if e.client != nil {
		for _, mutator := range requestMutators(ctx) {
			mutator(req)
		}
		return e.client.Request(ctx, req, e.opts...)
	}
	if e.server != nil {
		for _, mutator := range requestMutators(ctx) {
			mutator(req)
		}
		return e.server.Request(ctx, req)
	}
	return nil, errors.Errorf("unable to find client/server for connection ID %s", e.conn.GetId())
}

// Close - calls the begin.Close(ctx,conn) from the beginning of the chain.
// Uses the last networkservice.Connection successfully returned from the chain
// Note: if a chain is a server chain continued by a client chain, the beginning of the chain is at the beginning of
// the server chain, even if there is a subsequent begin.NewClient() in the client chain.
func (e *EventOriginator) Close(ctx context.Context) (*empty.Empty, error) {
	if e == nil {
		return nil, errors.Errorf("EventOriginator.Close should not be calle on a nil receiver")
	}
	if e.client != nil {
		return e.client.Close(ctx, e.conn, e.opts...)
	}
	if e.server != nil {
		return e.server.Close(ctx, e.conn)
	}
	return nil, errors.Errorf("unable to find EventOriginator.{client,server} for connection ID %s", e.conn.GetId())
}

func (e *EventOriginator) update(req *networkservice.NetworkServiceRequest, conn *networkservice.Connection, opts ...grpc.CallOption) {
	if e == nil {
		return
	}
	e.conn = conn.Clone()
	req.Connection = e.conn
	e.req = req
	e.opts = opts
}
