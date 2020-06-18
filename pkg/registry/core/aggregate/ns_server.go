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

type nsAggregateClient struct {
	grpc.ClientStream
	ctx     context.Context
	cancel  func()
	clients []registry.NetworkServiceRegistry_FindClient
	once    sync.Once
	ch      chan *registry.NetworkService
}

func (c *nsAggregateClient) initMonitoring() {
	for i := 0; i < len(c.clients); i++ {
		client := c.clients[i]
		go func() {
			for ns := range registry.ReadNetworkServiceChannel(client) {
				c.ch <- ns
			}
		}()
	}
}

func (c *nsAggregateClient) Recv() (*registry.NetworkService, error) {
	c.once.Do(c.initMonitoring)
	v, ok := <-c.ch
	if !ok {
		c.cancel()
		return nil, io.EOF
	}
	return v, nil
}

func (c *nsAggregateClient) Context() context.Context {
	return c.ctx
}

// NewNetworkServiceFindClient aggregates few NetworkServiceRegistry_FindClient to single  NetworkServiceRegistry_FindClient
func NewNetworkServiceFindClient(clients ...registry.NetworkServiceRegistry_FindClient) registry.NetworkServiceRegistry_FindClient {
	r := &nsAggregateClient{
		clients: clients,
		ch:      make(chan *registry.NetworkService),
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r
}
