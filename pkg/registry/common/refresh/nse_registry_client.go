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
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type refreshNSEClient struct {
	client                    registry.NetworkServiceEndpointRegistryClient
	nsesMutex                 sync.Mutex
	nseCancels                map[string]context.CancelFunc
	retryDelay                time.Duration
	defaultExpirationDuration time.Duration
}

func (c *refreshNSEClient) setDefaultExpiration(duration time.Duration) {
	c.defaultExpirationDuration = duration
}

func (c *refreshNSEClient) setRetryPeriod(p time.Duration) {
	c.retryDelay = p
}

func (c *refreshNSEClient) startRefresh(ctx context.Context, nse *registry.NetworkServiceEndpoint) {
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
				nse, err = c.client.Register(ctx, nse)
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
	resp, err := next.NetworkServiceEndpointRegistryClient(ctx).Register(ctx, in, opts...)
	if err != nil {
		return nil, err
	}
	c.nsesMutex.Lock()
	defer c.nsesMutex.Unlock()
	if resp.ExpirationTime == nil {
		expirationTime := time.Now().Add(c.defaultExpirationDuration)
		resp.ExpirationTime = &timestamp.Timestamp{
			Seconds: expirationTime.Unix(),
			Nanos:   int32(expirationTime.Nanosecond()),
		}
	}
	if v, ok := c.nseCancels[resp.Name]; ok {
		v()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.nseCancels[resp.Name] = cancel
	c.startRefresh(ctx, resp)
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
	c.nsesMutex.Lock()
	defer c.nsesMutex.Unlock()
	cancel, ok := c.nseCancels[in.Name]
	if ok {
		cancel()
		delete(c.nseCancels, in.Name)
	}
	return resp, nil
}

// NewNetworkServiceEndpointRegistryClient creates new NetworkServiceEndpointRegistryClient that will refresh expiration time for registered NSEs
func NewNetworkServiceEndpointRegistryClient(client registry.NetworkServiceEndpointRegistryClient, options ...Option) registry.NetworkServiceEndpointRegistryClient {
	c := &refreshNSEClient{
		client:                    client,
		nseCancels:                map[string]context.CancelFunc{},
		retryDelay:                time.Second * 5,
		defaultExpirationDuration: time.Minute * 30,
	}

	for _, o := range options {
		o.apply(c)
	}

	return c
}
