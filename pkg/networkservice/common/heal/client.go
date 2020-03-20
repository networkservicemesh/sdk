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
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/trace"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/heal"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type healClient struct {
	chainContext context.Context
	onHeal       *networkservice.NetworkServiceClient
	requestors   map[string]func()
	closers      map[string]func()
	executor     serialize.Executor
}

// NewClient - creates a new networkservice.NetworkServiceClient chain element that implements the healing algorithm
//             - ctx    - context for the lifecycle of the *Client* itself.  Cancel when discarding the client.
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
//			   - healer - *heal.Healer. Used to return healClient itself but in heal.Healer form.
//				  	      Since heal.Healer is an interface (and thus a pointer) *heal.Healer is a double pointer.
//						  Meaning it points to a place that points to a place that implements heal.Healer
//                        This is done so that we can return a networkservice.NetworkServiceClient chain element
//                        while maintaining the NewClient pattern for use like anything else in a chain.
//                        The value in *healer must be included in the monitor.NetworkServiceClient
//                        so it can call it to start healing of broken connections.
func NewClient(ctx context.Context, onHeal *networkservice.NetworkServiceClient, healer *heal.Healer) networkservice.NetworkServiceClient {
	rv := &healClient{
		chainContext: ctx,
		onHeal:       onHeal,
		requestors:   make(map[string]func()),
		closers:      make(map[string]func()),
		executor:     serialize.Executor{},
	}
	if rv.onHeal == nil {
		rv.onHeal = addressof.NetworkServiceClient(rv)
	}

	*healer = rv
	return rv
}

func (f *healClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	rv, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	// Clone the request
	req := request.Clone()
	// Set its connection to the returned connection we received
	req.Connection = rv

	// TODO handle deadline err
	deadline, _ := ctx.Deadline()
	duration := time.Until(deadline)
	f.executor.AsyncExec(func() {
		f.requestors[req.GetConnection().GetId()] = func() {
			timeCtx, cancelFunc := context.WithTimeout(context.Background(), duration)
			ctx = extend.WithValuesFromContext(timeCtx, ctx)
			// TODO wrap another span around this
			_, err := (*f.onHeal).Request(ctx, req, opts...)
			if err != nil {
				trace.Log(ctx).Errorf("Attempt to heal connection %s resulted in error: %+v", req.GetConnection().GetId(), err)
			}
			cancelFunc()
		}
		// TODO functions in closers map are never used at all!!!
		f.closers[req.GetConnection().GetId()] = func() {
			timeCtx, cancelFunc := context.WithTimeout(context.Background(), duration)
			ctx = extend.WithValuesFromContext(timeCtx, ctx)
			_, err := (*f.onHeal).Close(extend.WithValuesFromContext(timeCtx, ctx), req.GetConnection(), opts...)
			if err != nil {
				trace.Log(ctx).Errorf("Attempt to close connection %s during heal resulted in error: %+v", req.GetConnection().GetId(), err)
			}
			cancelFunc()
		}
	})
	return rv, nil
}

func (f *healClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	rv, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "Error calling next")
	}
	f.executor.AsyncExec(func() {
		delete(f.requestors, conn.GetId())
		delete(f.closers, conn.GetId())
	})
	return rv, nil
}

func (f *healClient) StartHeal(connectionID string) {
	if f, ok := f.requestors[connectionID]; ok {
		f()
	}
}
