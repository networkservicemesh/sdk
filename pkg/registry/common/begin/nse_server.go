// Copyright (c) 2022 Cisco and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type beginNSEServer struct {
	nseServerMap
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

		ctx = withEventFactory(ctx, eventFactoryServer)
		resp, err = next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, in)
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
		eventFactoryServer.updateContext(ctx)
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
		// If we don't have a connection to Close, just let it be
		return &emptypb.Empty{}, nil
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
		ctx = withEventFactory(ctx, eventFactoryServer)
		_, err = next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, eventFactoryServer.registration)
		eventFactoryServer.afterCloseFunc()
	})
	return &emptypb.Empty{}, err
}

// NewNetworkServiceEndpointRegistryServer - returns a new null server that does nothing but call next.NetworkServiceEndpointRegistryServer(ctx).
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return new(beginNSEServer)
}
