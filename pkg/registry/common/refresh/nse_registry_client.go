// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
)

type refreshNSEClient struct {
	chainContext          context.Context
	retryDelay            time.Duration
	defaultExpiryDuration time.Duration
	nseCancels            cancelsMap
}

func (c *refreshNSEClient) startRefresh(
	ctx context.Context,
	nextClient registry.NetworkServiceEndpointRegistryClient,
	nse *registry.NetworkServiceEndpoint,
	opts ...grpc.CallOption,
) {
	logEntry := log.Entry(ctx).WithField("refreshNSEClient", "startRefresh")

	t := nse.ExpirationTime.AsTime().Local()
	delta := time.Until(nse.ExpirationTime.AsTime().Local())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Until(t) / 3):
				nse.ExpirationTime = timestamppb.New(time.Now().Add(delta))

				r, err := nextClient.Register(ctx, nse, opts...)
				for err != nil {
					logEntry.Warnf("failed to refresh registration: %s", nse.Name)
					select {
					case <-ctx.Done():
						return
					case <-time.After(c.retryDelay):
						r, err = nextClient.Register(ctx, nse, opts...)
					}
				}

				nse = r
				t = nse.ExpirationTime.AsTime().Local()
			}
		}
	}()
}

func (c *refreshNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if in.ExpirationTime == nil {
		in.ExpirationTime = timestamppb.New(time.Now().Add(c.defaultExpiryDuration))
	}
	nse := in.Clone()

	nextClient := next.NetworkServiceEndpointRegistryClient(ctx)

	resp, err := nextClient.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	nse.ExpirationTime = resp.ExpirationTime

	if cancel, ok := c.nseCancels.Load(resp.Name); ok {
		cancel()
	}

	ctx, cancel := context.WithCancel(c.chainContext)
	c.nseCancels.Store(resp.Name, cancel)
	c.startRefresh(ctx, nextClient, nse, opts...)

	return resp, err
}

func (c *refreshNSEClient) Find(ctx context.Context, in *registry.NetworkServiceEndpointQuery, opts ...grpc.CallOption) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, in, opts...)
}

func (c *refreshNSEClient) Unregister(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*empty.Empty, error) {
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	if cancel, ok := c.nseCancels.LoadAndDelete(in.Name); ok {
		cancel()
	}

	return resp, nil
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(options ...Option) registry.NetworkServiceEndpointRegistryClient {
	c := &refreshNSEClient{
		chainContext:          context.Background(),
		retryDelay:            time.Second * 5,
		defaultExpiryDuration: time.Minute * 30,
	}

	for _, o := range options {
		o.apply(c)
	}

	return c
}
