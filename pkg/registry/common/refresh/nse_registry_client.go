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

package refresh

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/common/begin"
	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/clock"
)

type refreshNSEClient struct {
	ctx context.Context
	cancelsMap
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration
// time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(ctx context.Context) registry.NetworkServiceEndpointRegistryClient {
	return &refreshNSEClient{
		ctx: ctx,
	}
}

func (c *refreshNSEClient) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	var factory = begin.FromContext(ctx)

	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, nse, opts...)

	if err != nil {
		return nil, err
	}

	refreshCtx, cancel := context.WithCancel(c.ctx)

	if cancelPrevious, ok := c.LoadAndDelete(nse.Name); ok {
		cancelPrevious()
	}

	c.Store(nse.Name, cancel)

	var clockTime = clock.FromContext(ctx)

	if resp.GetExpirationTime() != nil {
		var refreshCh = clockTime.After(2 * clockTime.Until(resp.GetExpirationTime().AsTime().Local()) / 3)

		go func() {
			select {
			case <-refreshCtx.Done():
				return
			case <-refreshCh:
				<-factory.Register(begin.CancelContext(refreshCtx))
			}
		}()
	}

	return resp, err
}

func (c *refreshNSEClient) Find(ctx context.Context, query *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *refreshNSEClient) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	if v, ok := c.LoadAndDelete(nse.GetName()); ok {
		v()
	}
	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
