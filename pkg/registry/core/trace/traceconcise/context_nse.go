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
)

const (
	conciseNseInfoKey contextKeyType = "conciseNseInfoLogRegistry"
)

type conciseNseInfo struct {
	nseClientRegisterTail   registry.NetworkServiceEndpointRegistryClient
	nseClientFindHead       registry.NetworkServiceEndpointRegistry_FindClient
	nseClientFindTail       registry.NetworkServiceEndpointRegistry_FindClient
	nseClientUnregisterTail registry.NetworkServiceEndpointRegistryClient

	nseClientRegisterError   error
	nseClientFindError       error
	nseClientUnregisterError error
	nseClientRecvError       error

	nseServerRegisterTail   registry.NetworkServiceEndpointRegistryServer
	nseServerFindHead       registry.NetworkServiceEndpointRegistry_FindServer
	nseServerFindTail       registry.NetworkServiceEndpointRegistry_FindServer
	nseServerUnregisterTail registry.NetworkServiceEndpointRegistryServer

	nseServerRegisterError   error
	nseServerFindError       error
	nseServerUnregisterError error
	nseServerSendError       error
}

func conciseNseInfoFromCtx(ctx context.Context) (context.Context, *conciseNseInfo) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseNseInfoKey).(*conciseNseInfo)
	if ok {
		return ctx, v
	}
	v = new(conciseNseInfo)
	return context.WithValue(ctx, conciseNseInfoKey, v), v
}

func withNseClientRegisterTail(ctx context.Context, client registry.NetworkServiceEndpointRegistryClient) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseClientRegisterTail = client
	return c
}

func nseClientRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryClient) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseClientRegisterTail
}

func loadAndStoreNseClientRegisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseClientRegisterError
	d.nseClientRegisterError = err
	return prevErr
}

func withNseClientFindHead(ctx context.Context, client registry.NetworkServiceEndpointRegistry_FindClient) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseClientFindHead = client
	return c
}

func nseClientFindHead(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindClient) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseClientFindHead
}

func withNseClientFindTail(ctx context.Context, client registry.NetworkServiceEndpointRegistry_FindClient) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseClientFindTail = client
	return c
}

func nseClientFindTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindClient) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseClientFindTail
}

func loadAndStoreNseClientFindError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseClientFindError
	d.nseClientFindError = err
	return prevErr
}

func withNseClientUnregisterTail(ctx context.Context, client registry.NetworkServiceEndpointRegistryClient) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseClientUnregisterTail = client
	return c
}

func nseClientUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryClient) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseClientUnregisterTail
}

func loadAndStoreNseClientUnregisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseClientUnregisterError
	d.nseClientUnregisterError = err
	return prevErr
}

func loadAndStoreNseClientRecvError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseClientRecvError
	d.nseClientRecvError = err
	return prevErr
}

func withNseServerRegisterTail(ctx context.Context, server registry.NetworkServiceEndpointRegistryServer) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseServerRegisterTail = server
	return c
}

func nseServerRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryServer) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseServerRegisterTail
}

func loadAndStoreNseServerRegisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseServerRegisterError
	d.nseServerRegisterError = err
	return prevErr
}

func withNseServerFindHead(ctx context.Context, server registry.NetworkServiceEndpointRegistry_FindServer) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseServerFindHead = server
	return c
}

func nseServerFindHead(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindServer) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseServerFindHead
}

func withNseServerFindTail(ctx context.Context, server registry.NetworkServiceEndpointRegistry_FindServer) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseServerFindTail = server
	return c
}

func nseServerFindTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindServer) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseServerFindTail
}

func loadAndStoreNseServerFindError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseServerFindError
	d.nseServerFindError = err
	return prevErr
}

func withNseServerUnregisterTail(ctx context.Context, server registry.NetworkServiceEndpointRegistryServer) context.Context {
	c, d := conciseNseInfoFromCtx(ctx)
	d.nseServerUnregisterTail = server
	return c
}

func nseServerUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryServer) {
	c, d := conciseNseInfoFromCtx(ctx)
	return c, d.nseServerUnregisterTail
}

func loadAndStoreNseServerUnregisterError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseServerUnregisterError
	d.nseServerUnregisterError = err
	return prevErr
}

func loadAndStoreNseServerSendError(ctx context.Context, err error) (prevErr error) {
	_, d := conciseNseInfoFromCtx(ctx)
	prevErr = d.nseServerSendError
	d.nseServerSendError = err
	return prevErr
}
