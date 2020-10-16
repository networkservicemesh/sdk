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

package adapters

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/networkservice"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/networkservice/core/next"
)

const serverContextKey contextKeyType = "server"

type (
	serverToClient struct {
		server networkservice.NetworkServiceServer
	}
	ctxServer     struct{}
	nextClientCtx struct {
		next networkservice.NetworkServiceClient
		old  *nextClientCtx
	}
)

// NewServerToClient - returns a new networkservice.NetworkServiceClient that is a wrapper around server
func NewServerToClient(server networkservice.NetworkServiceServer) networkservice.NetworkServiceClient {
	return &serverToClient{server: next.NewNetworkServiceServer(server, &ctxServer{})}
}

func (s *serverToClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, _ ...grpc.CallOption) (*networkservice.Connection, error) {
	old, _ := ctx.Value(serverContextKey).(*nextClientCtx)
	nextCtx := context.WithValue(ctx, serverContextKey, &nextClientCtx{next: next.Client(ctx), old: old})
	return s.server.Request(nextCtx, in)
}

func (s *serverToClient) Close(ctx context.Context, in *networkservice.Connection, _ ...grpc.CallOption) (*empty.Empty, error) {
	old, _ := ctx.Value(serverContextKey).(*nextClientCtx)
	nextCtx := context.WithValue(ctx, serverContextKey, &nextClientCtx{next: next.Client(ctx), old: old})
	return s.server.Close(nextCtx, in)
}

func (s *ctxServer) Request(ctx context.Context, in *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	ctx2 := ctx.Value(serverContextKey).(*nextClientCtx)
	return ctx2.next.Request(context.WithValue(ctx, serverContextKey, ctx2.old), in)
}

func (s *ctxServer) Close(ctx context.Context, in *networkservice.Connection) (*empty.Empty, error) {
	ctx2 := ctx.Value(serverContextKey).(*nextClientCtx)
	return ctx2.next.Close(context.WithValue(ctx, serverContextKey, ctx2.old), in)
}
