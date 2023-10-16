// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

package begin

import (
	"context"
	"sync/atomic"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type queue struct {
	eventCount *atomic.Int64
	queueChan  chan func()
}

type beginNSEServer struct {
	genericsync.Map[string, *eventNSEFactoryServer]
	queueMap genericsync.Map[string, *queue]
}

func (b *beginNSEServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	id := in.GetName()
	if id == "" {
		return nil, errors.New("NetworkServiceEndpoint.Name can not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
	}
	eventFactoryServer, _ := b.LoadOrStore(id,
		newNSEEventFactoryServer(
			ctx,
			func() {
				b.Delete(id)
			},
		),
	)

	var resp *registry.NetworkServiceEndpoint
	var err error

	<-eventFactoryServer.executor.AsyncExec(func() {
		currentEventFactoryServer, _ := b.Load(id)
		if currentEventFactoryServer != eventFactoryServer {
			log.FromContext(ctx).Debug("recalling begin.Request because currentEventFactoryServer != eventFactoryServer")
			resp, err = b.Register(ctx, in)
			return
		}

		withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
		resp, err = next.NetworkServiceEndpointRegistryServer(withEventFactoryCtx).Register(withEventFactoryCtx, in)
		if err != nil {
			if eventFactoryServer.state != established {
				eventFactoryServer.state = closed
				b.Delete(id)
			}
			return
		}
		eventFactoryServer.registration = mergeNSE(in, resp)
		eventFactoryServer.state = established
		eventFactoryServer.response = resp
		eventFactoryServer.updateContext(grpcmetadata.PathWithContext(ctx, grpcmetadata.PathFromContext(ctx).Clone()))
	})
	return resp, err
}

func (b *beginNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (b *beginNSEServer) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	id := in.GetName()
	// 	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
	}
	eventFactoryServer, ok := b.Load(id)
	if !ok {
		q, loaded := b.queueMap.LoadOrStore(id, &queue{eventCount: &atomic.Int64{}, queueChan: make(chan func())})
		if !loaded {
			log.FromContext(ctx).Infof("Haven't found a queue. Starting a new one: %v", q)
			go func() {
				for event := range q.queueChan {
					log.FromContext(ctx).Infof("Got a new event")
					q.eventCount.Add(-1)
					event()

					if q.eventCount.Load() == int64(0) {
						log.FromContext(ctx).Infof("All events have been processed. Closing event factory...")
						b.queueMap.Delete(id)
						close(q.queueChan)
						return
					}
				}
			}()
		}

		q.eventCount.Add(1)
		waitCtx, cancel := context.WithCancel(ctx)
		var err error
		q.queueChan <- func() {
			_, err = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
			cancel()
		}

		<-waitCtx.Done()
		return &emptypb.Empty{}, err
	}
	var err error
	<-eventFactoryServer.executor.AsyncExec(func() {
		if eventFactoryServer.state != established || eventFactoryServer.registration == nil {
			return
		}
		currentServerClient, _ := b.Load(id)
		if currentServerClient != eventFactoryServer {
			return
		}
		withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
		_, err = next.NetworkServiceEndpointRegistryServer(withEventFactoryCtx).Unregister(withEventFactoryCtx, eventFactoryServer.registration)
		eventFactoryServer.afterCloseFunc()
	})
	return &emptypb.Empty{}, err
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(beginNSEServer)
}
