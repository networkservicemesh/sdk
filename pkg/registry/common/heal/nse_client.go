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

type healNSEClient struct {
	ctx context.Context
	cancelsMap
}

// NewNetworkServiceEndpointRegistryClient returns a new NSE registry client responsible for healing
func NewNetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	return &healNSEClient{
		ctx: ctx,
	}
}

func (c *healNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse.Clone(), opts...)

	if err != nil {
		return nil, err
	}

	factory := begin.FromContext(ctx)

	if v, ok := c.LoadAndDelete(nse.GetName()); ok {
		v()
	}
	healCtx, cancel := context.WithCancel(c.ctx)

	stream, streamErr := next.NetworkServiceEndpointRegistryClient(ctx).Find(healCtx, &registry.NetworkServiceEndpointQuery{NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{Name: nse.GetName()}, Watch: true}, opts...)

	if streamErr != nil {
		cancel()
		return nil, streamErr
	}

	c.Store(nse.GetName(), cancel)

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

func (c *healNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	if !query.Watch || isNSEFindHealing(ctx) {
		return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
	}

	query = proto.Clone(query).(*registry.NetworkServiceEndpointQuery)

	nextClient := next.NetworkServiceEndpointRegistryClient(ctx)

	createStream := func() (registry.NetworkServiceEndpointRegistry_FindClient, error) {
		queryClone := proto.Clone(query).(*registry.NetworkServiceEndpointQuery)
		return nextClient.Find(withNSEFindHealing(ctx), queryClone, opts...)
	}

	queryClone := proto.Clone(query).(*registry.NetworkServiceEndpointQuery)
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

	return &healNSEFindClient{
		ctx:          clientCtx,
		createStream: createStream,
		NetworkServiceEndpointRegistry_FindClient: stream,
	}, nil
}

func (c *healNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if v, loaded := c.LoadAndDelete(nse.Name); loaded {
		v()
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
