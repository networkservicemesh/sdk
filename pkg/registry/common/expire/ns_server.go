// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

package expire

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsServer struct {
	nseClient  registry.NetworkServiceEndpointRegistryClient
	monitorErr error
	timers     timerMap
	nsCounts   intMap
	contexts   contextMap
	once       sync.Once
	ctx        context.Context
}

func (n *nsServer) checkUpdates() {
	c, err := n.nseClient.Find(n.ctx, &registry.NetworkServiceEndpointQuery{
		NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
		Watch:                  true,
	})
	if err != nil {
		n.monitorErr = err
		return
	}
	for nse := range registry.ReadNetworkServiceEndpointChannel(c) {
		for i := range nse.NetworkServiceNames {
			ns := nse.NetworkServiceNames[i]
			var value int32
			stored, _ := n.nsCounts.LoadOrStore(ns, &value)
			if nse.ExpirationTime != nil && nse.ExpirationTime.Seconds < 0 {
				atomic.AddInt32(stored, -1)
			} else {
				atomic.AddInt32(stored, 1)
			}
			duration := time.Until(time.Unix(nse.ExpirationTime.Seconds, int64(nse.ExpirationTime.Nanos)))
			timer := time.AfterFunc(duration, func() {
				if atomic.AddInt32(stored, -1) <= 0 {
					if ctx, ok := n.contexts.Load(ns); ok {
						_, _ = n.Unregister(ctx, &registry.NetworkService{Name: ns})
					}
				}
			})
			if v, loaded := n.timers.LoadOrStore(nse.Name, timer); loaded {
				timer.Stop()
				v.Reset(duration)
			}
		}
	}
}

func (n *nsServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	n.once.Do(func() {
		go func() {
			for n.monitorErr == nil && n.ctx.Err() == nil {
				n.checkUpdates()
			}
		}()
	})
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	n.contexts.Store(request.Name, ctx)
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, request)
}

func (n *nsServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if n.monitorErr != nil {
		return n.monitorErr
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (n *nsServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	if v, ok := n.nsCounts.Load(request.Name); ok {
		if atomic.LoadInt32(v) > 0 {
			return nil, errors.New("cannot delete network service: resource already in use")
		}
	}
	n.contexts.Delete(request.Name)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceServer wraps passed NetworkServiceRegistryServer and monitor NetworkServiceEndpoints via passed NetworkServiceEndpointRegistryClient
func NewNetworkServiceServer(ctx context.Context, nseClient registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceRegistryServer {
	return &nsServer{nseClient: nseClient, ctx: ctx}
}
