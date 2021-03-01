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

package expire

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/extend"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type expireNSServer struct {
	nseClient  registry.NetworkServiceEndpointRegistryClient
	monitorErr error
	once       sync.Once
	chainCtx   context.Context
	nsStates   nsStateMap
}

type nsState struct {
	Timers  map[string]*time.Timer
	Context context.Context
	sync.Mutex
}

func (n *expireNSServer) checkUpdates(eventCh <-chan *registry.NetworkServiceEndpoint) {
	for event := range eventCh {
		nse := event
		if nse.ExpirationTime == nil {
			continue
		}

		for i := 0; i < len(nse.NetworkServiceNames); i++ {
			ns := nse.NetworkServiceNames[i]
			state, ok := n.nsStates.Load(ns)
			if !ok {
				continue
			}
			state.Lock()
			timer, ok := state.Timers[nse.Name]
			expirationDuration := time.Until(nse.ExpirationTime.AsTime().Local())
			if !ok {
				if expirationDuration > 0 {
					state.Timers[nse.Name] = time.AfterFunc(expirationDuration, func() {
						state.Lock()
						ctx := state.Context
						delete(state.Timers, nse.Name)
						state.Unlock()
						_, _ = n.Unregister(ctx, &registry.NetworkService{Name: ns})
					})
				}
				state.Unlock()
				continue
			}

			if expirationDuration < 0 {
				delete(state.Timers, nse.Name)
				if timer.Stop() {
					ctx := state.Context
					state.Unlock()
					_, _ = n.Unregister(ctx, &registry.NetworkService{Name: ns})
					continue
				}
				state.Unlock()
				continue
			}

			timer.Stop()
			timer.Reset(expirationDuration)
			state.Unlock()
		}
	}
}

func (n *expireNSServer) Register(ctx context.Context, request *registry.NetworkService) (*registry.NetworkService, error) {
	n.once.Do(func() {
		c, err := n.nseClient.Find(n.chainCtx, &registry.NetworkServiceEndpointQuery{
			NetworkServiceEndpoint: &registry.NetworkServiceEndpoint{},
			Watch:                  true,
		})
		if err != nil {
			n.monitorErr = err
			return
		}
		go n.checkUpdates(registry.ReadNetworkServiceEndpointChannel(c))
	})

	if n.monitorErr != nil {
		return nil, n.monitorErr
	}

	resp, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, request)

	if err != nil {
		return nil, err
	}

	valuesCtx := extend.WithValuesFromContext(n.chainCtx, ctx)

	v, _ := n.nsStates.LoadOrStore(request.Name, &nsState{
		Timers: make(map[string]*time.Timer),
	})

	v.Lock()
	v.Context = valuesCtx
	v.Unlock()

	return resp, nil
}

func (n *expireNSServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	if n.monitorErr != nil {
		return n.monitorErr
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (n *expireNSServer) Unregister(ctx context.Context, request *registry.NetworkService) (*empty.Empty, error) {
	if n.monitorErr != nil {
		return nil, n.monitorErr
	}

	state, ok := n.nsStates.Load(request.Name)

	if ok {
		state.Lock()
		if len(state.Timers) > 0 {
			state.Unlock()
			return nil, errors.New("cannot delete network service: resource already in use")
		}
		n.nsStates.Delete(request.Name)
		state.Unlock()
	}

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceServer wraps passed NetworkServiceRegistryServer and monitor NetworkServiceEndpoints via passed NetworkServiceEndpointRegistryClient
func NewNetworkServiceServer(ctx context.Context, nseClient registry.NetworkServiceEndpointRegistryClient) registry.NetworkServiceRegistryServer {
	return &expireNSServer{nseClient: nseClient, chainCtx: ctx}
}
