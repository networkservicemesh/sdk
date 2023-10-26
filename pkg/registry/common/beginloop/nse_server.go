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

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/common/grpcmetadata"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

func WithID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, "index", id)
}

func GetID(ctx context.Context) int {
	id, _ := ctx.Value("index").(int)
	return id
}

type customFactory struct {
	*eventNSEFactoryServer
	ch chan struct{}
}

type beginNSEServer struct {
	genericsync.Map[string, *customFactory]
}

func (b *beginNSEServer) Register(ctx context.Context, in *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	id := in.GetName()
	if id == "" {
		return nil, errors.New("NetworkServiceEndpoint.Name can not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	// if fromContext(ctx) != nil {
	// 	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
	// }

	var eventFactoryServer *customFactory
	for {
		var eventCount int
		var loaded bool
		eventFactoryServer, loaded = b.LoadOrStore(id, &customFactory{
			eventNSEFactoryServer: newNSEEventFactoryServer(ctx, 1, nil),
			ch:                    make(chan struct{}, 1)})

		if loaded {
			eventFactoryServer.ch <- struct{}{}
			log.FromContext(ctx).Infof("Thread [%v] loaded factory", in.Url)

			currentEventFactoryServer, _ := b.Load(id)
			if eventFactoryServer != currentEventFactoryServer {
				log.FromContext(ctx).Infof("Thread [%v] is doing continue", in.Url)
				<-eventFactoryServer.ch
				continue
			}

			eventFactoryServer.eventCount++
			eventCount = eventFactoryServer.eventCount
			log.FromContext(ctx).Infof("Thread [%v] finished to execute", in.Url)
			<-eventFactoryServer.ch
		}

		if eventCount > 0 || !loaded {
			log.FromContext(ctx).Infof("Thread [%v] exited loop", in.Url)
			break
		}
	}

	log.FromContext(ctx).Infof("Thread [%v] is trying to Lock channel", in.Url)
	eventFactoryServer.ch <- struct{}{}
	log.FromContext(ctx).Infof("Thread [%v] started main NEXT", in.Url)
	withEventFactoryCtx := withEventFactory(ctx, eventFactoryServer)
	resp, err := next.NetworkServiceEndpointRegistryServer(withEventFactoryCtx).Register(withEventFactoryCtx, in)
	eventFactoryServer.eventCount--
	if err != nil {
		<-eventFactoryServer.ch
		return nil, err
	}
	eventFactoryServer.registration = mergeNSE(in, resp)
	eventFactoryServer.state = established
	eventFactoryServer.response = resp
	eventFactoryServer.updateContext(grpcmetadata.PathWithContext(ctx, grpcmetadata.PathFromContext(ctx).Clone()))
	<-eventFactoryServer.ch
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
			eventNSEFactoryServer: newNSEEventFactoryServer(ctx, 1, nil),
			ch:                    make(chan struct{}, 1)})

		if loaded {
			eventFactoryServer.ch <- struct{}{}
			currentEventFactoryServer, _ := b.Load(id)
			if eventFactoryServer != currentEventFactoryServer {
				<-eventFactoryServer.ch
				continue
			}

			eventFactoryServer.eventCount++
			<-eventFactoryServer.ch
			break
		}

		if eventCount > 0 || !loaded {
			break
		}
	}

	eventFactoryServer.ch <- struct{}{}
	_, err := next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, in)
	eventFactoryServer.eventCount--
	if eventFactoryServer.eventCount == 0 {
		b.Delete(id)
	}
	<-eventFactoryServer.ch
	return &emptypb.Empty{}, err
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(beginNSEServer)
}
