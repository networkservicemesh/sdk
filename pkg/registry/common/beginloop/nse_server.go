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

package beginloop

import (
	"context"
	"sync"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

func WithID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, "index", id)
}

func GetID(ctx context.Context) int {
	id, _ := ctx.Value("index").(int)
	return id
}

type customFactory struct {
	sync.Mutex
	*eventNSEFactoryServer
}

type beginNSEServer struct {
	genericsync.Map[string, *customFactory]

	channel chan struct{}
}

func (b *beginNSEServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	// th := begin.GetID(ctx)
	// log.FromContext(ctx).Infof("ThreadID: %v", th)
	id := in.GetName()
	if id == "" {
		return nil, errors.New("NetworkServiceEndpoint.Name can not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
	}

	var eventFactoryServer *customFactory
	for {
		var eventCount int
		var loaded bool
		eventFactoryServer, loaded = b.LoadOrStore(id, &customFactory{
			eventNSEFactoryServer: newNSEEventFactoryServer(ctx, 1, nil)})

		if loaded {
			eventFactoryServer.Lock()
			currentEventFactoryServer, _ := b.Load(id)
			if eventFactoryServer != currentEventFactoryServer {
				eventFactoryServer.Unlock()
				continue
			}

			eventFactoryServer.eventCount++
			eventCount = eventFactoryServer.eventCount
			eventFactoryServer.Unlock()
		}

		if eventCount > 0 || !loaded {
			break
		}
	}

	eventFactoryServer.Lock()
	//b.channel <- struct{}{}
	withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
	resp, err := next.NetworkServiceEndpointRegistryServer(withEventFactoryCtx).Register(withEventFactoryCtx, in)
	eventFactoryServer.eventCount--
	if err != nil {
		eventFactoryServer.Unlock()
		return nil, err
	}
	eventFactoryServer.registration = mergeNSE(in, resp)
	eventFactoryServer.state = established
	eventFactoryServer.response = resp
	eventFactoryServer.updateContext(grpcmetadata.PathWithContext(ctx, grpcmetadata.PathFromContext(ctx).Clone()))
	//<-b.channel
	eventFactoryServer.Unlock()
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

	var eventFactoryServer *customFactory
	for {
		var eventCount int
		var loaded bool
		eventFactoryServer, loaded = b.LoadOrStore(id, &customFactory{
			eventNSEFactoryServer: newNSEEventFactoryServer(ctx, 1, nil)})

		if loaded {
			eventFactoryServer.Lock()
			currentEventFactoryServer, _ := b.Load(id)
			if eventFactoryServer != currentEventFactoryServer {
				eventFactoryServer.Unlock()
				continue
			}

			eventFactoryServer.eventCount++
			eventFactoryServer.Unlock()
			break
		}

		if eventCount > 0 || !loaded {
			break
		}
	}

	eventFactoryServer.Lock()
	_, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
	eventFactoryServer.eventCount--
	if eventFactoryServer.eventCount == 0 {
		b.Delete(id)
	}
	eventFactoryServer.Unlock()
	return &emptypb.Empty{}, err
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	s := &beginNSEServer{
		channel: make(chan struct{}, 1),
	}

	return s
}
