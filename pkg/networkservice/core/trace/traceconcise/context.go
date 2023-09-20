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

// Package traceconcise provides a wrapper for logging around a networkservice.NetworkServiceClient
package traceconcise

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

type contextKeyType string

const (
	conciseInfoKey contextKeyType = "conciseInfoLogNetworkservice"
	loggedType     string         = "networkService"
)

type conciseInfo struct {
	serverRequestTail *networkservice.NetworkServiceServer
	serverCloseTail   *networkservice.NetworkServiceServer
	clientRequestTail *networkservice.NetworkServiceClient
	clientCloseTail   *networkservice.NetworkServiceClient

	serverRequestError *error
	serverCloseError   *error
	clientRequestError *error
	clientCloseError   *error
}

func withLog(parent context.Context, connectionID, methodName string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	// Update outgoing grpc context
	parent = grpcutils.PassTraceToOutgoing(parent)

	fields := []*log.Field{log.NewField("id", connectionID), log.NewField("type", loggedType)}

	ctx, sLogger, span, sFinish := spanlogger.FromContext(parent, "", methodName, fields)
	ctx, lLogger, lFinish := logruslogger.FromSpan(ctx, span, "", fields)
	ctx = log.WithLog(ctx, sLogger, lLogger)

	return ctx, func() {
		sFinish()
		lFinish()
	}
}

func conciseInfoFromCtx(ctx context.Context) (context.Context, *conciseInfo) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseInfoKey).(*conciseInfo)
	if ok {
		return ctx, v
	}
	v = new(conciseInfo)
	return context.WithValue(ctx, conciseInfoKey, v), v
}

func withServerRequestTail(ctx context.Context, server *networkservice.NetworkServiceServer) context.Context {
	c, d := conciseInfoFromCtx(ctx)
	d.serverRequestTail = server
	return c
}

func serverRequestTail(ctx context.Context) (context.Context, *networkservice.NetworkServiceServer) {
	c, d := conciseInfoFromCtx(ctx)
	return c, d.serverRequestTail
}

func withServerCloseTail(ctx context.Context, server *networkservice.NetworkServiceServer) context.Context {
	c, d := conciseInfoFromCtx(ctx)
	d.serverCloseTail = server
	return c
}

func serverCloseTail(ctx context.Context) (context.Context, *networkservice.NetworkServiceServer) {
	c, d := conciseInfoFromCtx(ctx)
	return c, d.serverCloseTail
}

func withClientRequestTail(ctx context.Context, client *networkservice.NetworkServiceClient) context.Context {
	c, d := conciseInfoFromCtx(ctx)
	d.clientRequestTail = client
	return c
}

func clientRequestTail(ctx context.Context) (context.Context, *networkservice.NetworkServiceClient) {
	c, d := conciseInfoFromCtx(ctx)
	return c, d.clientRequestTail
}

func withClientCloseTail(ctx context.Context, client *networkservice.NetworkServiceClient) context.Context {
	c, d := conciseInfoFromCtx(ctx)
	d.clientCloseTail = client
	return c
}

func clientCloseTail(ctx context.Context) (context.Context, *networkservice.NetworkServiceClient) {
	c, d := conciseInfoFromCtx(ctx)
	return c, d.clientCloseTail
}

func loadAndStoreServerRequestError(ctx context.Context, err *error) (prevErr *error) {
	_, d := conciseInfoFromCtx(ctx)
	prevErr = d.serverRequestError
	d.serverRequestError = err
	return prevErr
}

func loadAndStoreServerCloseError(ctx context.Context, err *error) (prevErr *error) {
	_, d := conciseInfoFromCtx(ctx)
	prevErr = d.serverCloseError
	d.serverCloseError = err
	return prevErr
}

func loadAndStoreClientRequestError(ctx context.Context, err *error) (prevErr *error) {
	_, d := conciseInfoFromCtx(ctx)
	prevErr = d.clientRequestError
	d.clientRequestError = err
	return prevErr
}

func loadAndStoreClientCloseError(ctx context.Context, err *error) (prevErr *error) {
	_, d := conciseInfoFromCtx(ctx)
	prevErr = d.clientCloseError
	d.clientCloseError = err
	return prevErr
}
