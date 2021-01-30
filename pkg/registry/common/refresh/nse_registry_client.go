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
	"github.com/networkservicemesh/sdk/pkg/tools/logger"
)

type refreshNSEClient struct {
	chainContext          context.Context
	nseCancels            cancelsMap
	defaultExpiryDuration time.Duration
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration
// time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(ctx context.Context, options ...Option) registry.NetworkServiceEndpointRegistryClient {
	c := &refreshNSEClient{
		defaultExpiryDuration: time.Minute * 30,
		chainContext:          logger.WithLog(ctx),
	}

	for _, o := range options {
		o.apply(c)
	}

	return c
}

func (c *refreshNSEClient) startRefresh(
	ctx context.Context,
	client registry.NetworkServiceEndpointRegistryClient,
	nse *registry.NetworkServiceEndpoint,
) {
	logEntry := logger.Log(ctx).WithField("refreshNSEClient", "startRefresh")

	t := time.Unix(nse.ExpirationTime.Seconds, int64(nse.ExpirationTime.Nanos))
	delta := time.Until(t)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Until(t) / 3):
				nse.ExpirationTime = timestamppb.New(time.Now().Add(delta))

				res, err := client.Register(ctx, nse.Clone())
				if err != nil {
					logEntry.Errorf("failed to update registration: %s", err.Error())
					return
				}

				nse.ExpirationTime = res.ExpirationTime

				t = nse.ExpirationTime.AsTime().Local()
			}
		}
	}()
}

func (c *refreshNSEClient) Register(
	ctx context.Context,
	nse *registry.NetworkServiceEndpoint,
	opts ...grpc.CallOption,
) (*registry.NetworkServiceEndpoint, error) {
	if nse.ExpirationTime == nil {
		nse.ExpirationTime = timestamppb.New(time.Now().Add(c.defaultExpiryDuration))
	}

	refreshNSE := nse.Clone()

	nextClient := next.NetworkServiceEndpointRegistryClient(ctx)

	resp, err := nextClient.Register(ctx, nse, opts...)
	if err != nil {
		return nil, err
	}
	if cancel, ok := c.nseCancels.Load(resp.Name); ok {
		cancel()
	}

	refreshNSE.Name = resp.Name
	refreshNSE.ExpirationTime = resp.ExpirationTime

	ctx, cancel := context.WithCancel(c.chainContext)
	c.nseCancels.Store(resp.Name, cancel)

	c.startRefresh(ctx, nextClient, refreshNSE)

	return resp, err
}

func (c *refreshNSEClient) Find(
	ctx context.Context,
	query *registry.NetworkServiceEndpointQuery,
	opts ...grpc.CallOption,
) (registry.NetworkServiceEndpointRegistry_FindClient, error) {
	return next.NetworkServiceEndpointRegistryClient(ctx).Find(ctx, query, opts...)
}

func (c *refreshNSEClient) Unregister(
	ctx context.Context,
	nse *registry.NetworkServiceEndpoint,
	opts ...grpc.CallOption,
) (*empty.Empty, error) {
	if cancel, ok := c.nseCancels.Load(nse.Name); ok {
		cancel()
	}
	c.nseCancels.Delete(nse.Name)

	return next.NetworkServiceEndpointRegistryClient(ctx).Unregister(ctx, nse, opts...)
}
