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

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type queryCacheNSEClient struct {
	ctx   context.Context
	cache *cache
}

// NewClient creates new querycache NSE registry client that caches all resolved NSEs.
func NewClient(ctx context.Context, opts ...Option) registry.NetworkServiceEndpointRegistryClient {
	return &queryCacheNSEClient{
		ctx:   ctx,
		cache: newCache(ctx, opts...),
	}
}

func (q *queryCacheNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (q *queryCacheNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if query.GetWatch() {
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

	resultCh := make(chan *registry.NetworkServiceEndpointResponse, len(nses))
	for _, nse := range nses {
		resultCh <- &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: nse}
		q.storeInCache(ctx, nse.Clone(), opts...)
	}
	close(resultCh)

	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), nil
}

func (q *queryCacheNSEClient) findInCache(ctx context.Context, key string) (registry.NetworkServiceEndpointRegistry_FindClient, bool) {
	nse, ok := q.cache.Load(key)
	if !ok {
		return nil, false
	}

	resultCh := make(chan *registry.NetworkServiceEndpointResponse, 1)
	resultCh <- &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: nse.Clone()}
	close(resultCh)

	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), true
}

func (q *queryCacheNSEClient) storeInCache(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) {
	nseQuery := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
			Name: nse.GetName(),
		},
	}

	key := nseQuery.String()

	findCtx, cancel := context.WithCancel(q.ctx)

	entry, loaded := q.cache.LoadOrStore(key, nse, cancel)
	if loaded {
		cancel()
		return
	}

	go func() {
		defer entry.Cleanup()

		nseQuery.Watch = true

		stream, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(findCtx, nseQuery, opts...)
		if err != nil {
			return
		}

		for nseResp, err := stream.Recv(); err == nil; nseResp, err = stream.Recv() {
			if nseResp.GetNetworkServiceEndpoint().GetName() != nseQuery.GetNetworkServiceEndpoint().GetName() {
				continue
			}
			if nseResp.GetDeleted() {
				break
			}

			entry.Update(nseResp.GetNetworkServiceEndpoint())
		}
	}()
}

func (q *queryCacheNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}
