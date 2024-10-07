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

// Package querycache adds possibility to cache Find queries
package querycache

import (
	"context"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/registry/core/streamchannel"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type queryCacheNSClient struct {
	ctx   context.Context
	cache *nsCache
}

// NewNetworkServiceClient creates new querycache NS registry client that caches all resolved NSs
func NewNetworkServiceClient(ctx context.Context, opts ...Option) registry.NetworkServiceRegistryClient {
	return &queryCacheNSClient{
		ctx:   ctx,
		cache: newNSCache(ctx, opts...),
	}
}

func (q *queryCacheNSClient) Register(ctx context.Context, nse *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	return next.NetworkServiceRegistryClient(ctx).Register(ctx, nse, opts...)
}

func (q *queryCacheNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	log.FromContext(ctx).WithField("time", time.Now()).Infof("queryCacheNSClient forth")
	if query.Watch {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	}

	log.FromContext(ctx).WithField("time", time.Now()).Info("queryCacheNSClient search in cache")
	if client, ok := q.findInCache(ctx, query); ok {
		log.FromContext(ctx).Info("queryCacheNSClient found in cache")
		return client, nil
	}

	log.FromContext(ctx).WithField("time", time.Now()).Info("queryCacheNSClient not found in cache")

	client, err := next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	nses := registry.ReadNetworkServiceList(client)

	resultCh := make(chan *registry.NetworkServiceResponse, len(nses))
	for _, nse := range nses {
		resultCh <- &registry.NetworkServiceResponse{NetworkService: nse}
		q.storeInCache(ctx, nse.Clone(), opts...)
	}
	close(resultCh)

	return streamchannel.NewNetworkServiceFindClient(ctx, resultCh), nil
}

func (q *queryCacheNSClient) findInCache(ctx context.Context, query *registry.NetworkServiceQuery) (registry.NetworkServiceRegistry_FindClient, bool) {
	log.FromContext(ctx).WithField("time", time.Now()).Infof("queryCacheNSClient checking key: %v", query.NetworkService)
	ns := q.cache.Load(ctx, query.NetworkService)
	if ns == nil {
		return nil, false
	}

	log.FromContext(ctx).WithField("time", time.Now()).Infof("found NS in cache: %v", ns)

	resultCh := make(chan *registry.NetworkServiceResponse, 1)
	resultCh <- &registry.NetworkServiceResponse{NetworkService: ns.Clone()}
	close(resultCh)

	return streamchannel.NewNetworkServiceFindClient(ctx, resultCh), true
}

func (q *queryCacheNSClient) storeInCache(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) {
	nsQuery := &registry.NetworkServiceQuery{
		NetworkService: &registry.NetworkService{
			Name: ns.Name,
		},
	}

	findCtx, cancel := context.WithCancel(q.ctx)

	entry, loaded := q.cache.LoadOrStore(ns, cancel)
	if loaded {
		cancel()
		return
	}

	go func() {
		defer entry.Cleanup()

		nsQuery.Watch = true

		stream, err := next.NetworkServiceRegistryClient(ctx).Find(findCtx, nsQuery, opts...)
		if err != nil {
			return
		}

		for nsResp, err := stream.Recv(); err == nil; nsResp, err = stream.Recv() {
			if nsResp.NetworkService.Name != nsQuery.NetworkService.Name {
				continue
			}
			if nsResp.Deleted {
				break
			}

			entry.Update(nsResp.NetworkService)
		}
	}()
}

func (q *queryCacheNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}
