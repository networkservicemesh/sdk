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

// Package aggregate provides a possible to aggregate few stream clients to single interface
package aggregate

import (
	"context"
	"io"
	"sync"

	"github.com/networkservicemesh/api/pkg/api/registry"
	"google.golang.org/grpc"
)

type nseAggregateClient struct {
	grpc.ClientStream
	ctx     context.Context
	cancel  func()
	clients []registry.NetworkServiceEndpointRegistry_FindClient
	once    sync.Once
	ch      chan *registry.NetworkServiceEndpoint
}

func (c *nseAggregateClient) initMonitoring() {
	for i := 0; i < len(c.clients); i++ {
		client := c.clients[i]
		go func() {
			for ns := range registry.ReadNetworkServiceEndpointChannel(client) {
				c.ch <- ns
			}
		}()
	}
}

func (c *nseAggregateClient) Recv() (*registry.NetworkServiceEndpoint, error) {
	c.once.Do(c.initMonitoring)
	v, ok := <-c.ch
	if !ok {
		c.cancel()
		return nil, io.EOF
	}
	return v, nil
}

func (c *nseAggregateClient) Context() context.Context {
	return c.ctx
}

// NewNetworkServiceEndpointFindClient aggregates few NetworkServiceRegistry_FindClient to single NewNetworkServiceEndpointFindClient
func NewNetworkServiceEndpointFindClient(clients ...registry.NetworkServiceEndpointRegistry_FindClient) registry.NetworkServiceEndpointRegistry_FindClient {
	r := &nseAggregateClient{
		clients: clients,
		ch:      make(chan *registry.NetworkServiceEndpoint),
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r
}
