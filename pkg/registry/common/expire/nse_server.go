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

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nseServer struct {
	nses     map[string]*registry.NetworkServiceEndpoint
	executor serialize.Executor
	period   time.Duration
	once     sync.Once
	server   registry.NetworkServiceEndpointRegistryServer
}

func (n *nseServer) setPeriod(d time.Duration) {
	n.period = d
}

func (n *nseServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	n.once.Do(func() {
		n.server = next.NetworkServiceEndpointRegistryServer(ctx)
		n.monitor()
	})
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	n.executor.AsyncExec(func() {
		n.nses[nse.Name] = r
	})
	return r, nil
}

func (n *nseServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return n.server.Find(query, s)
}

func (n *nseServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	resp, err := n.server.Unregister(ctx, nse)
	if err != nil {
		return nil, err
	}
	n.executor.AsyncExec(func() {
		delete(n.nses, nse.Name)
	})
	return resp, nil
}

func (n *nseServer) monitor() {
	go func() {
		for {
			<-n.executor.AsyncExec(func() {
				for _, nse := range getExpiredNSEs(n.nses) {
					delete(n.nses, nse.Name)
					_, _ = n.server.Unregister(context.Background(), nse)
				}
			})
			<-time.After(n.period)
		}
	}()
}

// NewNetworkServiceEndpointRegistryServer wraps passed NetworkServiceEndpointRegistryServer and monitor Network service endpoints
func NewNetworkServiceEndpointRegistryServer(options ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &nseServer{
		period: defaultPeriod,
		nses:   map[string]*registry.NetworkServiceEndpoint{},
	}

	for _, o := range options {
		o.apply(r)
	}

	return r
}
