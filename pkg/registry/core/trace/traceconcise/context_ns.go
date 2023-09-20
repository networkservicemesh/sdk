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

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/log/spanlogger"
)

type contextKeyType string

const (
	conciseNsInfoKey contextKeyType = "conciseNsInfoLogRegistry"
	loggedType       string         = "registry"
)

type conciseNsInfo struct {
	nsClientRegisterTail   registry.NetworkServiceRegistryClient
	nsClientFindHead       registry.NetworkServiceRegistry_FindClient
	nsClientFindTail       registry.NetworkServiceRegistry_FindClient
	nsClientUnregisterTail registry.NetworkServiceRegistryClient

	nsClientRegisterError   error
	nsClientFindError       error
	nsClientUnregisterError error
	nsClientRecvError       error

	nsServerRegisterTail   registry.NetworkServiceRegistryServer
	nsServerFindHead       registry.NetworkServiceRegistry_FindServer
	nsServerFindTail       registry.NetworkServiceRegistry_FindServer
	nsServerUnregisterTail registry.NetworkServiceRegistryServer

	nsServerRegisterError   error
	nsServerFindError       error
	nsServerUnregisterError error
	nsServerSendError       error
}

// withLog - provides corresponding logger in context
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

	return ctx, func() {
		sFinish()
		lFinish()
	}
}

func conciseNsInfoFromCtx(ctx context.Context) (context.Context, *conciseNsInfo) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseNsInfoKey).(*conciseNsInfo)
	if ok {
		return ctx, v
	}
	v = new(conciseNsInfo)
	return context.WithValue(ctx, conciseNsInfoKey, v), v
}

func withNsClientRegisterTail(ctx context.Context, client registry.NetworkServiceRegistryClient) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsClientRegisterTail = client
	return c
}

func nsClientRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryClient) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsClientRegisterTail
}

func loadAndStoreNsClientRegisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsClientRegisterError
	d.nsClientRegisterError = err
	return prevErr
}

func withNsClientFindHead(ctx context.Context, client registry.NetworkServiceRegistry_FindClient) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsClientFindHead = client
	return c
}

func nsClientFindHead(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindClient) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsClientFindHead
}

func withNsClientFindTail(ctx context.Context, client registry.NetworkServiceRegistry_FindClient) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsClientFindTail = client
	return c
}

func nsClientFindTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindClient) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsClientFindTail
}

func loadAndStoreNsClientFindError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsClientFindError
	d.nsClientFindError = err
	return prevErr
}

func withNsClientUnregisterTail(ctx context.Context, client registry.NetworkServiceRegistryClient) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsClientUnregisterTail = client
	return c
}

func nsClientUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryClient) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsClientUnregisterTail
}

func loadAndStoreNsClientUnregisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsClientUnregisterError
	d.nsClientUnregisterError = err
	return prevErr
}

func loadAndStoreNsClientRecvError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsClientRecvError
	d.nsClientRecvError = err
	return prevErr
}

func withNsServerRegisterTail(ctx context.Context, server registry.NetworkServiceRegistryServer) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsServerRegisterTail = server
	return c
}

func nsServerRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryServer) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsServerRegisterTail
}

func loadAndStoreNsServerRegisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsServerRegisterError
	d.nsServerRegisterError = err
	return prevErr
}

func withNsServerFindHead(ctx context.Context, server registry.NetworkServiceRegistry_FindServer) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsServerFindHead = server
	return c
}

func nsServerFindHead(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindServer) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsServerFindHead
}

func withNsServerFindTail(ctx context.Context, server registry.NetworkServiceRegistry_FindServer) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsServerFindTail = server
	return c
}

func nsServerFindTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistry_FindServer) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsServerFindTail
}

func loadAndStoreNsServerFindError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsServerFindError
	d.nsServerFindError = err
	return prevErr
}

func withNsServerUnregisterTail(ctx context.Context, server registry.NetworkServiceRegistryServer) context.Context {
	c, d := conciseNsInfoFromCtx(ctx)
	d.nsServerUnregisterTail = server
	return c
}

func nsServerUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceRegistryServer) {
	c, d := conciseNsInfoFromCtx(ctx)
	return c, d.nsServerUnregisterTail
}

func loadAndStoreNsServerUnregisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsServerUnregisterError
	d.nsServerUnregisterError = err
	return prevErr
}

func loadAndStoreNsServerSendError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNsInfoFromCtx(ctx)
	prevErr = d.nsServerSendError
	d.nsServerSendError = err
	return prevErr
}
