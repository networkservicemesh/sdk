// Copyright (c) 2024 Cisco and/or its affiliates.
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

type queryCacheNSClient struct {
	chainContext context.Context
	cache        cache.Cache[string, []*registry.NetworkService]
}

// NewNetworkServiceRegistryClient creates new querycache NS registry client that caches all resolved NSs
func NewNetworkServiceRegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	var res = &queryCacheNSClient{
		chainContext: ctx,
		cache:        cache.NewCache[string, []*registry.NetworkService]().WithLRU().WithMaxKeys(32).WithTTL(time.Millisecond * 100),
	}
	return res
}

func (q *queryCacheNSClient) Register(ctx context.Context, nse *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	resp, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, nse, opts...)
	if err == nil {
		q.cache.Add(resp.GetName(), []*registry.NetworkService{resp})
	}
	return resp, err
}

func (q *queryCacheNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if query.Watch {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	}

	var list []*registry.NetworkService
	if v, ok := q.cache.Get(query.GetNetworkService().GetName()); ok {
		list = v
	} else {
		var streamClient, err = next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
		if err != nil {
			return streamClient, err
		}
		list = registry.ReadNetworkServiceList(streamClient)
		for _, item := range list {
			q.cache.Add(item.GetName(), []*registry.NetworkService{item.Clone()})
		}
	}
	var resultStreamChannel = make(chan *registry.NetworkServiceResponse, len(list))
	for _, item := range list {
		resultStreamChannel <- &registry.NetworkServiceResponse{NetworkService: item}
	}
	close(resultStreamChannel)
	return streamchannel.NewNetworkServiceFindClient(ctx, resultStreamChannel), nil
}

func (q *queryCacheNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	resp, err := next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
	if err == nil {
		q.cache.Remove(in.GetName())
	}
	return resp, err
}
