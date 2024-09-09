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

package traceconcise

import (
	"context"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
)

type contextKeyType string

type conciseMap = sync.Map

const (
	conciseMapNsKey contextKeyType = "conciseMapNsLogRegistry"
	traceInfoKey    contextKeyType = "traceInfoKeyRegistry"
	loggedType      string         = "registry"

	nsClientRegisterTailKey   = "nsClientRegisterTailKey"
	nsClientFindHeadKey       = "nsClientFindHeadKey"
	nsClientFindTailKey       = "nsClientFindTailKey"
	nsClientUnregisterTailKey = "nsClientUnregisterTailKey"

	nsClientRegisterErrorKey   = "nsClientRegisterError"
	nsClientFindErrorKey       = "nsClientFindError"
	nsClientUnregisterErrorKey = "nsClientUnregisterError"
	nsClientRecvErrorKey       = "nsClientRecvError"

	nsServerRegisterTailKey   = "nsServerRegisterTail"
	nsServerFindHeadKey       = "nsServerFindHead"
	nsServerFindTailKey       = "nsServerFindTail"
	nsServerUnregisterTailKey = "nsServerUnregisterTail"

	nsServerRegisterErrorKey   = "nsServerRegisterError"
	nsServerFindErrorKey       = "nsServerFindError"
	nsServerUnregisterErrorKey = "nsServerUnregisterError"
	nsServerSendErrorKey       = "nsServerSendError"
)

// withLog - provides corresponding logger in context.
func withLog(parent context.Context, methodName string) (c context.Context, f func()) {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	// Update outgoing grpc context
	parent = grpcutils.PassTraceToOutgoing(parent)

	fields := []*log.Field{log.NewField("type", loggedType)}

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

func conciseMapNsFromCtx(ctx context.Context) (context.Context, *conciseMap) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseMapNsKey).(*conciseMap)
	if ok {
		return ctx, v
	}
	v = new(conciseMap)
	return context.WithValue(ctx, conciseMapNsKey, v), v
}

func withNsClientRegisterTail(ctx context.Context, client registry.NetworkServiceRegistryClient) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsClientRegisterTailKey, client)
	return c
}

func nsClientRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryClient) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsClientRegisterTailKey); ok {
		return c, v.(registry.NetworkServiceRegistryClient)
	}
	return c, nil
}

func loadAndStoreNsClientRegisterError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsClientRegisterErrorKey)
	d.Store(nsClientRegisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNsClientFindHead(ctx context.Context, client registry.NetworkServiceRegistry_FindClient) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsClientFindHeadKey, client)
	return c
}

func nsClientFindHead(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindClient) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsClientFindHeadKey); ok {
		return c, v.(registry.NetworkServiceRegistry_FindClient)
	}
	return c, nil
}

func withNsClientFindTail(ctx context.Context, client registry.NetworkServiceRegistry_FindClient) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsClientFindTailKey, client)
	return c
}

func nsClientFindTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindClient) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsClientFindTailKey); ok {
		return c, v.(registry.NetworkServiceRegistry_FindClient)
	}
	return c, nil
}

func loadAndStoreNsClientFindError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsClientFindErrorKey)
	d.Store(nsClientFindErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNsClientUnregisterTail(ctx context.Context, client registry.NetworkServiceRegistryClient) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsClientUnregisterTailKey, client)
	return c
}

func nsClientUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryClient) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsClientUnregisterTailKey); ok {
		return c, v.(registry.NetworkServiceRegistryClient)
	}
	return c, nil
}

func loadAndStoreNsClientUnregisterError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsClientUnregisterErrorKey)
	d.Store(nsClientUnregisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreNsClientRecvError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsClientRecvErrorKey)
	d.Store(nsClientRecvErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNsServerRegisterTail(ctx context.Context, server registry.NetworkServiceRegistryServer) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsServerRegisterTailKey, server)
	return c
}

func nsServerRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryServer) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsServerRegisterTailKey); ok {
		return c, v.(registry.NetworkServiceRegistryServer)
	}
	return c, nil
}

func loadAndStoreNsServerRegisterError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsServerRegisterErrorKey)
	d.Store(nsServerRegisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNsServerFindHead(ctx context.Context, server registry.NetworkServiceRegistry_FindServer) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsServerFindHeadKey, server)
	return c
}

func nsServerFindHead(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindServer) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsServerFindHeadKey); ok {
		return c, v.(registry.NetworkServiceRegistry_FindServer)
	}
	return c, nil
}

func withNsServerFindTail(ctx context.Context, server registry.NetworkServiceRegistry_FindServer) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsServerFindTailKey, server)
	return c
}

func nsServerFindTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindServer) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsServerFindTailKey); ok {
		return c, v.(registry.NetworkServiceRegistry_FindServer)
	}
	return c, nil
}

func loadAndStoreNsServerFindError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsServerFindErrorKey)
	d.Store(nsServerFindErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNsServerUnregisterTail(ctx context.Context, server registry.NetworkServiceRegistryServer) context.Context {
	c, d := conciseMapNsFromCtx(ctx)
	d.Store(nsServerUnregisterTailKey, server)
	return c
}

func nsServerUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryServer) {
	c, d := conciseMapNsFromCtx(ctx)
	if v, ok := d.Load(nsServerUnregisterTailKey); ok {
		return c, v.(registry.NetworkServiceRegistryServer)
	}
	return c, nil
}

func loadAndStoreNsServerUnregisterError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nsServerUnregisterErrorKey)
	d.Store(nsServerUnregisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreNsServerSendError(ctx context.Context, err error) error {
	_, d := conciseMapNsFromCtx(ctx)

	prevErr, ok := d.Load(nseServerSendErrorKey)
	d.Store(nsServerSendErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}
