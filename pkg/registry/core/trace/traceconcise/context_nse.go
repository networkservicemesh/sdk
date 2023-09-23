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
	conciseMapNseKey contextKeyType = "conciseMapNseLogRegistry"

	nseClientRegisterTailKey   = "nseClientRegisterTailKey"
	nseClientFindHeadKey       = "nseClientFindHeadKey"
	nseClientFindTailKey       = "nseClientFindTailKey"
	nseClientUnregisterTailKey = "nseClientUnregisterTailKey"

	nseClientRegisterErrorKey   = "nseClientRegisterError"
	nseClientFindErrorKey       = "nseClientFindError"
	nseClientUnregisterErrorKey = "nseClientUnregisterError"
	nseClientRecvErrorKey       = "nseClientRecvError"

	nseServerRegisterTailKey   = "nseServerRegisterTail"
	nseServerFindHeadKey       = "nseServerFindHead"
	nseServerFindTailKey       = "nseServerFindTail"
	nseServerUnregisterTailKey = "nseServerUnregisterTail"

	nseServerRegisterErrorKey   = "nseServerRegisterError"
	nseServerFindErrorKey       = "nseServerFindError"
	nseServerUnregisterErrorKey = "nseServerUnregisterError"
	nseServerSendErrorKey       = "nseServerSendError"
)

func conciseMapNseFromCtx(ctx context.Context) (context.Context, *conciseMap) {
	if ctx == nil {
		panic("cannot create context from nil parent")
	}

	v, ok := ctx.Value(conciseMapNseKey).(*conciseMap)
	if ok {
		return ctx, v
	}
	v = new(conciseMap)
	return context.WithValue(ctx, conciseMapNseKey, v), v
}

func withNseClientRegisterTail(ctx context.Context, client registry.NetworkServiceEndpointRegistryClient) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseClientRegisterTailKey, client)
	return c
}

func nseClientRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryClient) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseClientRegisterTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistryClient)
	}
	return c, nil
}

func loadAndStoreNseClientRegisterError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseClientRegisterErrorKey)
	d.Store(nseClientRegisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNseClientFindHead(ctx context.Context, client registry.NetworkServiceEndpointRegistry_FindClient) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseClientFindHeadKey, client)
	return c
}

func nseClientFindHead(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindClient) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseClientFindHeadKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistry_FindClient)
	}
	return c, nil
}

func withNseClientFindTail(ctx context.Context, client registry.NetworkServiceEndpointRegistry_FindClient) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseClientFindTailKey, client)
	return c
}

func nseClientFindTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindClient) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseClientFindTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistry_FindClient)
	}
	return c, nil
}

func loadAndStoreNseClientFindError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseClientFindErrorKey)
	d.Store(nseClientFindErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNseClientUnregisterTail(ctx context.Context, client registry.NetworkServiceEndpointRegistryClient) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseClientUnregisterTailKey, client)
	return c
}

func nseClientUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryClient) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseClientUnregisterTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistryClient)
	}
	return c, nil
}

func loadAndStoreNseClientUnregisterError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseClientUnregisterErrorKey)
	d.Store(nseClientUnregisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreNseClientRecvError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseClientRecvErrorKey)
	d.Store(nseClientRecvErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNseServerRegisterTail(ctx context.Context, server registry.NetworkServiceEndpointRegistryServer) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseServerRegisterTailKey, server)
	return c
}

func nseServerRegisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryServer) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseServerRegisterTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistryServer)
	}
	return c, nil
}

func loadAndStoreNseServerRegisterError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseServerRegisterErrorKey)
	d.Store(nseServerRegisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNseServerFindHead(ctx context.Context, server registry.NetworkServiceEndpointRegistry_FindServer) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseServerFindHeadKey, server)
	return c
}

func nseServerFindHead(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindServer) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseServerFindHeadKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistry_FindServer)
	}
	return c, nil
}

func withNseServerFindTail(ctx context.Context, server registry.NetworkServiceEndpointRegistry_FindServer) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseServerFindTailKey, server)
	return c
}

func nseServerFindTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistry_FindServer) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseServerFindTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistry_FindServer)
	}
	return c, nil
}

func loadAndStoreNseServerFindError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseServerFindErrorKey)
	d.Store(nseServerFindErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func withNseServerUnregisterTail(ctx context.Context, server registry.NetworkServiceEndpointRegistryServer) context.Context {
	c, d := conciseMapNseFromCtx(ctx)
	d.Store(nseServerUnregisterTailKey, server)
	return c
}

func nseServerUnregisterTail(ctx context.Context) (context.Context, registry.NetworkServiceEndpointRegistryServer) {
	c, d := conciseMapNseFromCtx(ctx)
	if v, ok := d.Load(nseServerUnregisterTailKey); ok {
		return c, v.(registry.NetworkServiceEndpointRegistryServer)
	}
	return c, nil
}

func loadAndStoreNseServerUnregisterError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseServerUnregisterErrorKey)
	d.Store(nseServerUnregisterErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}

func loadAndStoreNseServerSendError(ctx context.Context, err error) error {
	_, d := conciseMapNseFromCtx(ctx)

	prevErr, ok := d.Load(nseServerSendErrorKey)
	d.Store(nseServerSendErrorKey, err)
	if ok {
		return prevErr.(error)
	}
	return nil
}
