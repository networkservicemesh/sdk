// Copyright (c) 2023 Cisco and/or its affiliates.
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

// Package retry provies a chain elemen that manages retries for failed requests
package retry

import (
	"context"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/common/begin"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

type cancelableContext struct {
	context.Context
	cancel func()
}

type retryClient struct {
	chainCtx   context.Context
	contextMap genericsync.Map[string, *cancelableContext]
}

// NewClient returns a new client that retries the request in case the previous attempt failed.
func NewClient(ctx context.Context) networkservice.NetworkServiceClient {
	return &retryClient{
		chainCtx: ctx,
	}
}

func (n *retryClient) Request(ctx context.Context, request *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	factory := begin.FromContext(ctx)
	cancelableCtx := new(cancelableContext)
	cancelableCtx.Context, cancelableCtx.cancel = context.WithCancel(n.chainCtx)
	cancelableCtx, _ = n.contextMap.LoadOrStore(request.GetConnection().GetId(), cancelableCtx)
	resp, err := next.Client(ctx).Request(ctx, request, opts...)
	if err != nil {
		var opts []begin.Option
		opts = append(opts, begin.CancelContext(cancelableCtx.Context))
		if request.GetConnection().GetNetworkServiceEndpointName() != "" && request.GetConnection().GetState() != networkservice.State_RESELECT_REQUESTED {
			opts = append(opts, begin.WithReselect())
		}
		factory.Request(opts...)
	} else {
		if v, ok := n.contextMap.LoadAndDelete(request.GetConnection().GetId()); ok {
			v.cancel()
		}
	}
	return resp, err
}

func (n *retryClient) Close(ctx context.Context, conn *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	resp, err := next.Client(ctx).Close(ctx, conn, opts...)
	if err == nil {
		if v, ok := n.contextMap.LoadAndDelete(conn.GetId()); ok {
			v.cancel()
		}
	}
	return resp, err
}
