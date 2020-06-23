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

package expire

import (
	"context"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nseServer struct {
	nses      map[string]*registry.NetworkServiceEndpoint
	nsesMutex sync.Mutex
	period    time.Duration
	once      sync.Once
	server    registry.NetworkServiceEndpointRegistryServer
}

func (n *nseServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	n.once.Do(n.monitor)
	n.nsesMutex.Lock()
	n.nses[nse.Name] = nse
	n.nsesMutex.Unlock()
	return n.server.Register(ctx, nse)
}

func (n *nseServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return n.server.Find(query, s)
}

func (n *nseServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	n.nsesMutex.Lock()
	delete(n.nses, nse.Name)
	n.nsesMutex.Unlock()
	return n.server.Unregister(ctx, nse)
}

func (n *nseServer) monitor() {
	go func() {
		for {
			n.nsesMutex.Lock()
			for _, nse := range removeExpiredNSEs(n.nses) {
				delete(n.nses, nse.Name)
				_, _ = n.server.Unregister(context.Background(), nse)
			}
			n.nsesMutex.Unlock()

			<-time.After(n.period)
		}
	}()
}

// NewNetworkServiceEndpointRegistryServer wraps passed NetworkServiceEndpointRegistryServer and monitor Network service endpoints
func NewNetworkServiceEndpointRegistryServer(server registry.NetworkServiceEndpointRegistryServer) registry.NetworkServiceEndpointRegistryServer {
	return &nseServer{
		server: server,
		period: 100 * time.Millisecond,
		nses:   map[string]*registry.NetworkServiceEndpoint{},
	}
}
