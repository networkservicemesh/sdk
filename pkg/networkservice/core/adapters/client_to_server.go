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

const clientContextKey contextKeyType = "client"

type (
	clientToServer struct {
		client networkservice.NetworkServiceClient
	}
	ctxClient     struct{}
	nextServerCtx struct {
		next networkservice.NetworkServiceServer
		old  *nextServerCtx
	}
)

// NewClientToServer - returns a networkservice.NetworkServiceServer wrapped around the supplied client
func NewClientToServer(client networkservice.NetworkServiceClient) networkservice.NetworkServiceServer {
	return &clientToServer{client: next.NewNetworkServiceClient(client, &ctxClient{})}
}

func (c *clientToServer) Request(ctx context.Context, request *networkservice.NetworkServiceRequest) (*networkservice.Connection, error) {
	old, _ := ctx.Value(clientContextKey).(*nextServerCtx)
	nextCtx := context.WithValue(ctx, clientContextKey, &nextServerCtx{next: next.Server(ctx), old: old})
	return c.client.Request(nextCtx, request)
}

func (c *clientToServer) Close(ctx context.Context, request *networkservice.Connection) (*empty.Empty, error) {
	old, _ := ctx.Value(clientContextKey).(*nextServerCtx)
	nextCtx := context.WithValue(ctx, clientContextKey, &nextServerCtx{next: next.Server(ctx), old: old})
	return c.client.Close(nextCtx, request)
}

func (c *ctxClient) Request(ctx context.Context, in *networkservice.NetworkServiceRequest, opts ...grpc.CallOption) (*networkservice.Connection, error) {
	ctx2 := ctx.Value(clientContextKey).(*nextServerCtx)
	return ctx2.next.Request(context.WithValue(ctx, clientContextKey, ctx2.old), in)
}

func (c *ctxClient) Close(ctx context.Context, in *networkservice.Connection, opts ...grpc.CallOption) (*empty.Empty, error) {
	ctx2 := ctx.Value(clientContextKey).(*nextServerCtx)
	return ctx2.next.Close(context.WithValue(ctx, clientContextKey, ctx2.old), in)
}
