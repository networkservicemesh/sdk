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
	"sync"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"

	"github.com/networkservicemesh/api/pkg/api/networkservice"

	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
)

type contextKeyType string

const (
	conciseMapKey contextKeyType = "conciseMapLogNetworkservice"
	traceInfoKey  contextKeyType = "traceInfoKeyNetworkservice"
	loggedType    string         = "networkService"

	serverRequestTailKey = "serverRequestTail"
	serverCloseTailKey   = "serverCloseTail"
	clientRequestTailKey = "clientRequestTail"
	clientCloseTailKey   = "clientCloseTail"

	serverRequestErrorKey = "serverRequestError"
	serverCloseErrorKey   = "serverCloseError"
	clientRequestErrorKey = "clientRequestError"
	clientCloseErrorKey   = "clientCloseError"
)

type conciseMap = sync.Map

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

	if grpcTraceState := grpcutils.TraceFromContext(parent); (grpcTraceState == grpcutils.TraceOn) ||
		(grpcTraceState == grpcutils.TraceUndefined && log.IsTracingEnabled()) {
		ctx = withTrace(ctx)
	}

	return ctx, func() {
		sFinish()
		lFinish()
	}
}

// withConnectionInfo - Provides a traceInfo in context.
func withTrace(parent context.Context) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	if ok := trace(parent); ok {
		// We already had connection info
		return parent
	}

	return context.WithValue(parent, traceInfoKey, &struct{}{})
}

// trace - return traceInfo from context.
func trace(ctx context.Context) bool {
	return ctx.Value(traceInfoKey) != nil
}

func conciseMapFromCtx(ctx context.Context) (context.Context, *conciseMap) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseMapKey).(*conciseMap)
	if ok {
		return ctx, v
	}
	v = new(conciseMap)
	return context.WithValue(ctx, conciseMapKey, v), v
}

func withServerRequestTail(ctx context.Context, server networkservice.NetworkServiceServer) context.Context {
	c, d := conciseMapFromCtx(ctx)
	d.Store(serverRequestTailKey, server)
	return c
}

func serverRequestTail(ctx context.Context) (context.Context, networkservice.NetworkServiceServer) {
	c, d := conciseMapFromCtx(ctx)
	if v, ok := d.Load(serverRequestTailKey); ok {
		return c, v.(networkservice.NetworkServiceServer)
	}
	return c, nil
}

func withServerCloseTail(ctx context.Context, server networkservice.NetworkServiceServer) context.Context {
	c, d := conciseMapFromCtx(ctx)
	d.Store(serverCloseTailKey, server)
	return c
}

func serverCloseTail(ctx context.Context) (context.Context, networkservice.NetworkServiceServer) {
	c, d := conciseMapFromCtx(ctx)
	if v, ok := d.Load(serverCloseTailKey); ok {
		return c, v.(networkservice.NetworkServiceServer)
	}
	return c, nil
}

func withClientRequestTail(ctx context.Context, client networkservice.NetworkServiceClient) context.Context {
	c, d := conciseMapFromCtx(ctx)
	d.Store(clientRequestTailKey, client)
	return c
}

func clientRequestTail(ctx context.Context) (context.Context, networkservice.NetworkServiceClient) {
	c, d := conciseMapFromCtx(ctx)
	if v, ok := d.Load(clientRequestTailKey); ok {
		return c, v.(networkservice.NetworkServiceClient)
	}
	return c, nil
}

func withClientCloseTail(ctx context.Context, client networkservice.NetworkServiceClient) context.Context {
	c, d := conciseMapFromCtx(ctx)
	d.Store(clientCloseTailKey, client)
	return c
}

func clientCloseTail(ctx context.Context) (context.Context, networkservice.NetworkServiceClient) {
	c, d := conciseMapFromCtx(ctx)
	if v, ok := d.Load(clientCloseTailKey); ok {
		return c, v.(networkservice.NetworkServiceClient)
	}
	return c, nil
}

func loadAndStoreServerRequestError(ctx context.Context, err error) error {
	_, d := conciseMapFromCtx(ctx)

	prevErr, ok := d.Load(serverRequestErrorKey)
	d.Store(serverRequestErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreServerCloseError(ctx context.Context, err error) error {
	_, d := conciseMapFromCtx(ctx)

	prevErr, ok := d.Load(serverCloseErrorKey)
	d.Store(serverCloseErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreClientRequestError(ctx context.Context, err error) error {
	_, d := conciseMapFromCtx(ctx)

	prevErr, ok := d.Load(clientRequestErrorKey)
	d.Store(clientRequestErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreClientCloseError(ctx context.Context, err error) error {
	_, d := conciseMapFromCtx(ctx)

	prevErr, ok := d.Load(clientCloseErrorKey)
	d.Store(clientCloseErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}
