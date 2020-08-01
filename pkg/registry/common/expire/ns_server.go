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
	"errors"
	"sync"
	"time"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type nsServer struct {
	server     registry.NetworkServiceRegistryServer
	nseClient  registry.NetworkServiceEndpointRegistryClient
	once       sync.Once
	period     time.Duration
	monitorErr error
	nses       map[string]*registry.NetworkServiceEndpoint
	nsCounter  map[string]int64
	nss        map[string]*registry.NetworkService
	executor   serialize.Executor
}

func (n *nsServer) setPeriod(d time.Duration) {
	n.period = d
}

func (n *nsServer) monitorUpdates() {
	for {
		c, err := n.nseClient.Find(context.Background(), &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
			Watch:                  true,
		})
		if err != nil {
			n.monitorErr = err
			return
		}
		for event := range registry.ReadNetworkServiceEndpointChannel(c) {
			nse := event
			n.executor.AsyncExec(func() {
				_, exist := n.nses[nse.Name]
				n.nses[nse.Name] = nse
				if !exist {
					for _, service := range nse.NetworkServiceNames {
						n.nsCounter[service]++
					}
				}
			})
		}
	}
}

func (n *nsServer) monitorNSEsExpiration() {
	for {
		var list []*registry.NetworkService
		n.executor.AsyncExec(func() {
			for _, nse := range getExpiredNSEs(n.nses) {
				for _, service := range nse.NetworkServiceNames {
					n.nsCounter[service]--
					if n.nsCounter[service] == 0 {
						ns, ok := n.nss[service]
						if ok {
							list = append(list, ns)
						}
					}
				}
			}
			for _, ns := range list {
				delete(n.nsCounter, ns.Name)
				delete(n.nss, ns.Name)
				_, _ = n.server.Unregister(context.Background(), ns)
			}
		})
		<-time.After(n.period)
	}
}

func (n *nsServer) monitor() {
	go func() {
		n.monitorUpdates()
	}()
	go func() {
		n.monitorNSEsExpiration()
	}()
}

func (n *nsServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	n.once.Do(func() {
		n.server = next.NetworkServiceRegistryServer(ctx)
		n.monitor()
	})
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	r, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, request)
	if err != nil {
		return nil, err
	}
	n.executor.AsyncExec(func() {
		n.nss[request.Name] = r
	})
	return r, nil
}

func (n *nsServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if n.monitorErr != nil {
		return n.monitorErr
	}
	return n.server.Find(query, s)
}

func (n *nsServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	canClose := false
	n.executor.AsyncExec(func() {
		if n.nsCounter[request.Name] == 0 {
			canClose = true
		}
	})
	if !canClose {
		return new(empty.Empty), errors.New("can not delete ns cause of already in use")
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceServer wraps passed NetworkServiceRegistryServer and monitor NetworkServiceEndpoints via passed NetworkServiceEndpointRegistryClient
func NewNetworkServiceServer(nseClient registry.NetworkServiceEndpointRegistryClient, options ...Option) registry.NetworkServiceRegistryServer {
	r := &nsServer{
		nseClient: nseClient,
		nsCounter: map[string]int64{},
		nss:       map[string]*registry.NetworkService{},
		nses:      map[string]*registry.NetworkServiceEndpoint{},
	}

	for _, o := range options {
		o.apply(r)
	}

	return r
}
