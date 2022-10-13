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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type beginNSClient struct {
	nsClientMap
}

func (b *beginNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	id := in.GetName()
	if id == "" {
		return nil, errors.New("registry.NetworkService.Name must not be zero valued")
	}
	// If some other EventFactory is already in the ctx... we are already running in an executor, and can just execute normally
	if fromContext(ctx) != nil {
		return next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
	}
	eventFactoryClient, _ := b.LoadOrStore(id,
		newEventNSFactoryClient(
			ctx,
			func() {
				b.Delete(id)
			},
			opts...,
		),
	)
	var resp *registry.NetworkService
	var err error
	<-eventFactoryClient.executor.AsyncExec(func() {
		// If the eventFactory has changed, usually because the connection has been Closed and re-established
		// go back to the beginning and try again.
		currentEventFactoryClient, _ := b.Load(id)
		if currentEventFactoryClient != eventFactoryClient {
			log.FromContext(ctx).Debug("recalling begin.Request because currentEventFactoryClient != eventFactoryClient")
			resp, err = b.Register(ctx, in, opts...)
			return
		}

		ctx = withEventFactory(ctx, eventFactoryClient)
		resp, err = next.NetworkServiceRegistryClient(ctx).Register(ctx, in, opts...)
		if err != nil {
			if eventFactoryClient.state != established {
				eventFactoryClient.state = closed
				b.Delete(id)
			}
			return
		}
		eventFactoryClient.opts = opts
		eventFactoryClient.state = established
		eventFactoryClient.registration = mergeNS(in, resp.Clone())
		eventFactoryClient.response = resp.Clone()
		eventFactoryClient.updateContext(ctx)
	})
	return resp, err
}

func (b *beginNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (b *beginNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	id := in.GetName()
	if fromContext(ctx) != nil {
		return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
	}
	eventFactoryClient, ok := b.Load(id)
	if !ok {
		return new(empty.Empty), nil
	}
	var emp *empty.Empty
	var err error
	<-eventFactoryClient.executor.AsyncExec(func() {
		// If the connection is not established, don't do anything
		if eventFactoryClient.state != established || eventFactoryClient.client == nil || eventFactoryClient.registration == nil {
			return
		}

		// If this isn't the connection we started with, do nothing
		currentEventFactoryClient, _ := b.Load(id)
		if currentEventFactoryClient != eventFactoryClient {
			return
		}
		// Always close with the last valid Connection we got
		ctx = withEventFactory(ctx, eventFactoryClient)
		emp, err = next.NetworkServiceRegistryClient(ctx).Unregister(ctx, eventFactoryClient.registration, opts...)
		// afterCloseFunc() is used to cleanup things like the entry in the Map for EventFactories
		eventFactoryClient.afterCloseFunc()
	})
	return emp, err
}

// NewNetworkServiceRegistryClient - returns a new null client that does nothing but call next.NetworkServiceRegistryClient(ctx).
func NewNetworkServiceRegistryClient() registry.NetworkServiceRegistryClient {
	return new(beginNSClient)
}
