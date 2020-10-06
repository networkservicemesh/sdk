// Copyright (c) 2020 Doc.ai and/or its affiliates.
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

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type refreshNSEClient struct {
	nseCancels            cancelsMap
	retryDelay            time.Duration
	defaultExpiryDuration time.Duration
}

func (c *refreshNSEClient) startRefresh(ctx context.Context, client registry.NetworkServiceEndpointRegistryClient, nse *registry.NetworkServiceEndpoint) {
	t := time.Unix(nse.ExpirationTime.Seconds, int64(nse.ExpirationTime.Nanos))
	delta := time.Until(t)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Until(t) / 3):
				t1 := time.Now().Add(delta)
				nse.ExpirationTime.Seconds = t1.Unix()
				nse.ExpirationTime.Nanos = int32(t1.Nanosecond())
				var err error
				nse, err = client.Register(ctx, nse)
				if err != nil {
					<-time.After(c.retryDelay)
					continue
				}
				t = t1
			}
		}
	}()
}

func (c *refreshNSEClient) Register(ctx context.Context, in *registry.NetworkServiceEndpoint, opts ...grpc.CallOption) (*registry.NetworkServiceEndpoint, error) {
	if in.ExpirationTime == nil {
		expirationTime := time.Now().Add(c.defaultExpiryDuration)
		in.ExpirationTime = &timestamp.Timestamp{
			Seconds: expirationTime.Unix(),
			Nanos:   int32(expirationTime.Nanosecond()),
		}
	}
	nextClient := next.NetworkServiceEndpointRegistryClient(ctx)
	resp, err := nextClient.Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	if cancel, ok := c.nseCancels.Load(resp.Name); ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.nseCancels.Store(resp.Name, cancel)
	c.startRefresh(ctx, nextClient, resp)
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
	if cancel, ok := c.nseCancels.Load(in.Name); ok {
		cancel()
	}
	c.nseCancels.Delete(in.Name)
	return resp, nil
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(options ...Option) registry.NetworkServiceEndpointRegistryClient {
	c := &refreshNSEClient{
		retryDelay:            time.Second * 5,
		defaultExpiryDuration: time.Minute * 30,
	}

	for _, o := range options {
		o.apply(c)
	}

	return c
}
