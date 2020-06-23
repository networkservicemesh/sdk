// Copyright (c) 2020 Cisco Systems, Inc.
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type nsServer struct {
	server     registry.NetworkServiceRegistryServer
	nseClient  registry.NetworkServiceEndpointRegistryClient
	once       sync.Once
	period     time.Duration
	monitorErr error
	nses       map[string]*registry.NetworkServiceEndpoint
	nsCounter  map[string]int
	nss        map[string]*registry.NetworkService
	executor   serialize.Executor
}

func (n *nsServer) monitor() {
	go func() {
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
				n.nses[nse.Name] = nse
				for _, service := range nse.NetworkServiceName {
					n.nsCounter[service]++
				}
			})
		}
	}()
	go func() {
		for {
			var list []*registry.NetworkService
			n.executor.AsyncExec(func() {
				for _, nse := range removeExpiredNSEs(n.nses) {
					for _, service := range nse.NetworkServiceName {
						n.nsCounter[service]--
						if n.nsCounter[service] == 0 {
							ns, ok := n.nss[service]
							if ok {
								list = append(list, ns)
							}
						}
					}
				}
			})

			n.executor.AsyncExec(func() {
				for _, ns := range list {
					delete(n.nsCounter, ns.Name)
					delete(n.nss, ns.Name)
					_, _ = n.server.Unregister(context.Background(), ns)
				}
			})
			<-time.After(n.period)
		}
	}()
}

func (n *nsServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	n.once.Do(n.monitor)
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}
	<-n.executor.AsyncExec(func() {
		n.nss[request.Name] = request
	})
	return n.server.Register(ctx, request)
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
	canDelete := false
	<-n.executor.AsyncExec(func() {
		n.nss[request.Name] = request
		if n.nsCounter[request.Name] == 0 {
			canDelete = true
		}
	})
	if canDelete {
		return n.server.Unregister(ctx, request)
	}
	return new(empty.Empty), errors.New("can not delete ns cause of already in use")
}

// NewNetworkServiceServer wraps passed NetworkServiceRegistryServer and monitor NetworkServiceEndpoints via passed NetworkServiceEndpointRegistryClient
func NewNetworkServiceServer(s registry.NetworkServiceRegistryServer, nseClient registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceRegistryServer {
	return &nsServer{
		server:    s,
		nseClient: nseClient,
		nsCounter: make(map[string]int),
		nss:       map[string]*registry.NetworkService{},
		nses:      map[string]*registry.NetworkServiceEndpoint{},
	}
}
