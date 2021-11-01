// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

package heal

import (
	"context"
	"sync"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/addressof"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type healNSEClient struct {
	ctx      context.Context
	onHeal   *registry.NetworkServiceEndpointRegistryClient
	nseInfos nseInfoMap

	stream     registry.NetworkServiceEndpointRegistry_FindClient
	healCancel context.CancelFunc
	lock       sync.RWMutex
}

type nseInfo struct {
	nse    *registry.NetworkServiceEndpoint
	ctx    context.Context
	cancel context.CancelFunc
}

// NewNetworkServiceEndpointRegistryClient returns a new NSE registry client responsible for healing
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, onHeal *registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceEndpointRegistryClient {
	c := &healNSEClient{
		ctx:        ctx,
		onHeal:     onHeal,
		healCancel: func() {},
	}
	if c.onHeal == nil {
		c.onHeal = addressof.NetworkServiceEndpointRegistryClient(c)
	}
	return c
}

func (c *healNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	nseCtx, nseCancel := context.WithCancel(c.ctx)
	_, loaded := c.nseInfos.LoadOrStore(nse.Name, &nseInfo{
		nse:    nse.Clone(),
		ctx:    nseCtx,
		cancel: nseCancel,
	})
	if loaded {
		nseCancel()
	}

	if err := c.startMonitor(ctx, opts); err != nil {
		return nil, err
	}

	reg, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)
	if err != nil {
		if !loaded {
			nseCancel()
			c.nseInfos.Delete(nse.Name)
		}
		return nil, err
	}

	return reg, nil
}

func (c *healNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if !query.Watch || isNSEFindHealing(ctx) {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	}

	query = proto.Clone(query).(*registry.NetworkServiceEndpointQuery)

	createStream := func() (registry.NetworkServiceEndpointRegistry_FindClient, error) {
		queryClone := proto.Clone(query).(*registry.NetworkServiceEndpointQuery)
		return (*c.onHeal).Find(withNSEFindHealing(ctx), queryClone, opts...)
	}

	queryClone := proto.Clone(query).(*registry.NetworkServiceEndpointQuery)
	stream, err := next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, queryClone, opts...)
	if err != nil {
		return nil, err
	}

	clientCtx, clientCancel := context.WithCancel(c.ctx)
	go func() {
		select {
		case <-clientCtx.Done():
		case <-ctx.Done():
		}
		clientCancel()
	}()

	return &healNSEFindClient{
		ctx:          clientCtx,
		createStream: createStream,
		NetworkServiceEndpointRegistry_FindClient: stream,
	}, nil
}

func (c *healNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	info, loaded := c.nseInfos.LoadAndDelete(nse.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	info.cancel()

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}

func (c *healNSEClient) startMonitor(ctx context.Context, opts []grpc.CallOption) error {
	logger := log.FromContext(c.ctx).WithField("healNSEClient", "startMonitor")

	c.lock.RLock()
	stream := c.stream
	c.lock.RUnlock()

	if stream != nil {
		return nil
	}

	c.lock.Lock()

	if c.stream != nil {
		c.lock.Unlock()
		return nil
	}

	findCtx, findCancel := context.WithCancel(c.ctx)
	findCtx = extend.WithValuesFromContext(findCtx, ctx)
	findCtx = log.WithFields(findCtx, map[string]interface{}{})

	query := &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: new(registry.NetworkServiceEndpoint),
		Watch:                  true,
	}

	var err error
	c.stream, err = next.NetworkServiceEndpointRegistryClient(ctx).Find(findCtx, query, opts...)

	c.lock.Unlock()

	if err != nil {
		logger.Warn("NSE client failed")
		findCancel()
		return err
	}

	logger.Info("NSE client ready")

	go func() {
		defer findCancel()
		c.monitor(opts)
	}()

	return nil
}

func (c *healNSEClient) monitor(opts []grpc.CallOption) {
	for _, err := c.stream.Recv(); err == nil; _, err = c.stream.Recv() {
	}
	c.healCancel()

	c.lock.Lock()
	defer c.lock.Unlock()

	c.restore(opts)
}

func (c *healNSEClient) restore(opts []grpc.CallOption) {
	log.FromContext(c.ctx).WithField("healNSEClient", "restore").Warn("NSE client restoring")

	c.stream = nil

	var healCtx context.Context
	healCtx, c.healCancel = context.WithCancel(c.ctx)

	c.nseInfos.Range(func(name string, info *nseInfo) bool {
		go func() {
			nseCtx, nseCancel := context.WithCancel(extend.WithValuesFromContext(healCtx, context.Background()))
			defer nseCancel()

			go func() {
				select {
				case <-nseCtx.Done():
				case <-info.ctx.Done():
				}
				nseCancel()
			}()

			for nseCtx.Err() == nil {
				if _, err := (*c.onHeal).Register(nseCtx, info.nse.Clone(), opts...); err == nil {
					return
				}
			}
		}()
		return true
	})
}
