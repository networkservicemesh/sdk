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

	"github.com/networkservicemesh/sdk/pkg/registry/common/memory"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type queryCacheNSEClient struct {
	chainCtx context.Context
	cache    memory.NetworkServiceEndpointSyncMap
}

func (q *queryCacheNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
}

func (q *queryCacheNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if in.Watch {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	}

	if nse, ok := q.cache.Load(in.String()); ok {
		resultCh := make(chan *registry.NetworkServiceEndpoint, 1)
		resultCh <- nse
		close(resultCh)
		return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), nil
	}

	client, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	nses := registry.ReadNetworkServiceEndpointList(client)

	resultCh := make(chan *registry.NetworkServiceEndpoint, len(nses))
	for _, nse := range nses {
		resultCh <- nse

		nseQuery := &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{
				Name: nse.Name,
			},
		}

		key := nseQuery.String()
		q.cache.Store(key, nse)

		go func() {
			defer q.cache.Delete(key)

			findCtx, findCancel := context.WithCancel(q.chainCtx)
			defer findCancel()

			nseQuery.Watch = true

			stream, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(findCtx, nseQuery, opts...)
			if err != nil {
				return
			}

			for update, err := stream.Recv(); err == nil; update, err = stream.Recv() {
				if update.Name != nseQuery.NetworkServiceEndpoint.Name {
					continue
				}
				if update.ExpirationTime != nil && update.ExpirationTime.Seconds < 0 {
					break
				}
				q.cache.Store(key, update)
			}
		}()
	}
	close(resultCh)

	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultCh), nil
}

func (q *queryCacheNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewClient creates new querycache registry.NetworkServiceEndpointRegistryClient that caches all resolved NSEs
// All cached NSE is
func NewClient(chainCtx context.Context) registry.NetworkServiceEndpointRegistryClient {
	return &queryCacheNSEClient{chainCtx: chainCtx}
}
