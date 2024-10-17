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
	"time"

	cache "github.com/go-pkgz/expirable-cache/v3"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
)

type queryCacheNSEClient struct {
	chainContext context.Context
	cache        cache.Cache[string, []*registry.NetworkServiceEndpoint]
}

// NewClient creates new querycache NSE registry client that caches all resolved NSEs
func NewNetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	var res = &queryCacheNSEClient{
		chainContext: ctx,
		cache:        cache.NewCache[string, []*registry.NetworkServiceEndpoint]().WithLRU().WithMaxKeys(32).WithTTL(time.Millisecond * 300),
	}
	return res
}

func (q *queryCacheNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err == nil {
		q.cache.Add(resp.GetName(), []*registry.NetworkServiceEndpoint{resp})
	}
	return resp, err
}

func (q *queryCacheNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if query.Watch {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	}

	var list []*registry.NetworkServiceEndpoint
	if v, ok := q.cache.Get(query.GetNetworkServiceEndpoint().GetName()); ok {
		list = v
	} else {
		var streamClient, err = next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
		if err != nil {
			return streamClient, err
		}
		list = registry.ReadNetworkServiceEndpointList(streamClient)
		for _, item := range list {
			q.cache.Add(item.GetName(), []*registry.NetworkServiceEndpoint{item.Clone()})
		}
	}
	var resultStreamChannel = make(chan *registry.NetworkServiceEndpointResponse, len(list))
	for _, item := range list {
		resultStreamChannel <- &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: item}
	}
	close(resultStreamChannel)
	return streamchannel.NewNetworkServiceEndpointFindClient(ctx, resultStreamChannel), nil
}

func (q *queryCacheNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
	if err == nil {
		q.cache.Remove(in.GetName())
	}
	return resp, err
}
