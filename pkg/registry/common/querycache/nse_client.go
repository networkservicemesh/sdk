// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

// Package querycache adds possible to cache Find queries
package querycache

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/after"
)

type queryCacheNSEClient struct {
	ctx           context.Context
	expireTimeout time.Duration
	cache         cacheEntryMap
}

type cacheEntry struct {
	nse       *registry.NetworkServiceEndpoint
	afterFunc *after.Func
	lock      sync.RWMutex
}

// NewClient creates new querycache NSE registry client that caches all resolved NSEs
func NewClient(ctx context.Context, expireTimeout time.Duration) registry.NetworkServiceEndpointRegistryClient {
	return &queryCacheNSEClient{
		ctx:           ctx,
		expireTimeout: expireTimeout,
	}
}

func (q *queryCacheNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (q *queryCacheNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if query.Watch {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	}

	if client, ok := q.findInCache(ctx, query.String()); ok {
		return client, nil
	}

	client, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	nses := registry.ReadNetworkServiceEndpointList(client)

	resultCh := make(chan *registry.NetworkServiceEndpoint, len(nses))
	for _, nse := range nses {
		resultCh <- nse
		q.storeInCache(ctx, nse, opts...)
	}
	close(resultCh)

	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), nil
}

func (q *queryCacheNSEClient) findInCache(ctx context.Context, key string) (registry.NetworkServiceEndpointRegistry_FindClient, bool) {
	entry, ok := q.cache.Load(key)
	if !ok {
		return nil, false
	}

	entry.lock.RLock()
	defer entry.lock.RUnlock()

	if !entry.afterFunc.Stop() {
		return nil, false
	}
	entry.afterFunc.Reset(time.Now().Add(q.expireTimeout))

	resultCh := make(chan *registry.NetworkServiceEndpoint, 1)
	resultCh <- entry.nse
	close(resultCh)

	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), true
}

func (q *queryCacheNSEClient) storeInCache(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) {
	nseQuery := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: nse.Name,
		},
	}

	key := nseQuery.String()

	findCtx, findCancel := context.WithCancel(q.ctx)
	entry := &cacheEntry{
		nse:       nse,
		afterFunc: after.NewFunc(findCtx, time.Now().Add(q.expireTimeout), findCancel),
	}
	q.cache.Store(key, entry)

	go func() {
		defer findCancel()
		defer func() {
			entry.lock.Lock()
			entry.afterFunc.Stop()
			entry.lock.Unlock()
		}()
		defer q.cache.Delete(key)

		nseQuery.Watch = true

		stream, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(findCtx, nseQuery, opts...)
		if err != nil {
			return
		}

		for nse, err = stream.Recv(); err == nil; nse, err = stream.Recv() {
			if nse.Name != nseQuery.NetworkServiceEndpoint.Name {
				continue
			}
			if nse.ExpirationTime != nil && nse.ExpirationTime.Seconds < 0 {
				break
			}

			entry.lock.Lock()
			entry.nse = nse
			entry.lock.Unlock()
		}
	}()
}

func (q *queryCacheNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}
