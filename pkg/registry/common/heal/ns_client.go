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

type healNSClient struct {
	ctx     context.Context
	onHeal  *registry.NetworkServiceRegistryClient
	nsInfos nsInfoMap

	stream     registry.NetworkServiceRegistry_FindClient
	healCancel context.CancelFunc
	lock       sync.RWMutex
}

type nsInfo struct {
	ns     *registry.NetworkService
	ctx    context.Context
	cancel context.CancelFunc
}

// NewNetworkServiceRegistryClient returns a new NS registry client responsible for healing
func NewNetworkServiceRegistryClient(ctx context.Context, onHeal *registry.NetworkServiceRegistryClient) registry.NetworkServiceRegistryClient {
	c := &healNSClient{
		ctx:        ctx,
		onHeal:     onHeal,
		healCancel: func() {},
	}
	if c.onHeal == nil {
		c.onHeal = addressof.NetworkServiceRegistryClient(c)
	}
	return c
}

func (c *healNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	nsCtx, nsCancel := context.WithCancel(c.ctx)
	_, loaded := c.nsInfos.LoadOrStore(ns.Name, &nsInfo{
		ns:     ns.Clone(),
		ctx:    nsCtx,
		cancel: nsCancel,
	})
	if loaded {
		nsCancel()
	}

	if err := c.startMonitor(ctx, opts); err != nil {
		return nil, err
	}

	reg, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns, opts...)
	if err != nil {
		if !loaded {
			nsCancel()
			c.nsInfos.Delete(ns.Name)
		}
		return nil, err
	}

	return reg, nil
}

func (c *healNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if !query.Watch || isNSFindHealing(ctx) {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	}

	query = proto.Clone(query).(*registry.NetworkServiceQuery)

	createStream := func() (registry.NetworkServiceRegistry_FindClient, error) {
		queryClone := proto.Clone(query).(*registry.NetworkServiceQuery)
		return (*c.onHeal).Find(withNSFindHealing(ctx), queryClone, opts...)
	}

	queryClone := proto.Clone(query).(*registry.NetworkServiceQuery)
	stream, err := next.NetworkServiceRegistryClient(ctx).Find(ctx, queryClone, opts...)
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

	return &healNSFindClient{
		ctx:                               clientCtx,
		createStream:                      createStream,
		NetworkServiceRegistry_FindClient: stream,
	}, nil
}

func (c *healNSClient) Unregister(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	info, loaded := c.nsInfos.LoadAndDelete(ns.Name)
	if !loaded {
		return new(empty.Empty), nil
	}

	info.cancel()

	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}

func (c *healNSClient) startMonitor(ctx context.Context, opts []grpc.CallOption) error {
	logger := log.FromContext(c.ctx).WithField("healNSClient", "startMonitor")

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

	query := &registry.NetworkServiceQuery{
		NetworkService: new(registry.NetworkService),
		Watch:          true,
	}

	var err error
	c.stream, err = next.NetworkServiceRegistryClient(ctx).Find(findCtx, query, opts...)

	c.lock.Unlock()

	if err != nil {
		logger.Warn("NS client failed")
		findCancel()
		return err
	}

	logger.Info("NS client ready")

	go func() {
		defer findCancel()
		c.monitor(opts)
	}()

	return nil
}

func (c *healNSClient) monitor(opts []grpc.CallOption) {
	for _, err := c.stream.Recv(); err == nil; _, err = c.stream.Recv() {
	}
	c.healCancel()

	c.lock.Lock()
	defer c.lock.Unlock()

	c.restore(opts)
}

func (c *healNSClient) restore(opts []grpc.CallOption) {
	log.FromContext(c.ctx).WithField("healNSClient", "restore").Warn("NS client restoring")

	c.stream = nil

	var healCtx context.Context
	healCtx, c.healCancel = context.WithCancel(c.ctx)

	c.nsInfos.Range(func(name string, info *nsInfo) bool {
		go func() {
			nsCtx, nsCancel := context.WithCancel(extend.WithValuesFromContext(healCtx, context.Background()))
			defer nsCancel()

			go func() {
				select {
				case <-nsCtx.Done():
				case <-info.ctx.Done():
				}
				nsCancel()
			}()

			for nsCtx.Err() == nil {
				if _, err := (*c.onHeal).Register(nsCtx, info.ns.Clone(), opts...); err == nil {
					return
				}
			}
		}()
		return true
	})
}
