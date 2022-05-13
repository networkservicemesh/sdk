// Copyright (c) 2021-2022 Doc.ai and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type healNSClient struct {
	ctx context.Context
	cancelsMap
}

// NewNetworkServiceRegistryClient returns a new NS registry client responsible for healing
func NewNetworkServiceRegistryClient(ctx context.Context) registry.NetworkServiceRegistryClient {
	return &healNSClient{
		ctx: ctx,
	}
}

func (c *healNSClient) Register(ctx context.Context, ns *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	resp, err := next.NetworkServiceRegistryClient(ctx).Register(ctx, ns.Clone(), opts...)

	if err != nil {
		return nil, err
	}

	factory := begin.FromContext(ctx)

	if v, ok := c.LoadAndDelete(ns.GetName()); ok {
		v()
	}
	healCtx, cancel := context.WithCancel(c.ctx)

	stream, streamErr := next.NetworkServiceRegistryClient(ctx).Find(healCtx, &registry.NetworkServiceQuery{NetworkService: &registry.NetworkService{Name: ns.GetName()}, Watch: true}, opts...)

	if streamErr != nil {
		cancel()
		return nil, streamErr
	}

	c.Store(ns.GetName(), cancel)

	go func() {
		for {
			_, recvErr := stream.Recv()
			if recvErr != nil {
				factory.Register(begin.CancelContext(healCtx))
				return
			}
		}
	}()

	return resp, err
}

func (c *healNSClient) Find(ctx context.Context, query *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	if !query.Watch || isNSFindHealing(ctx) {
		return next.NetworkServiceRegistryClient(ctx).Find(ctx, query, opts...)
	}

	query = proto.Clone(query).(*registry.NetworkServiceQuery)

	nextClient := next.NetworkServiceRegistryClient(ctx)

	createStream := func() (registry.NetworkServiceRegistry_FindClient, error) {
		queryClone := proto.Clone(query).(*registry.NetworkServiceQuery)
		return nextClient.Find(withNSFindHealing(ctx), queryClone, opts...)
	}

	queryClone := proto.Clone(query).(*registry.NetworkServiceQuery)
	stream, err := nextClient.Find(ctx, queryClone, opts...)
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
	if v, loaded := c.LoadAndDelete(ns.Name); loaded {
		v()
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, ns, opts...)
}
