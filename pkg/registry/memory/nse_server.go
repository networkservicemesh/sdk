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

package memory

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
	"github.com/networkservicemesh/sdk/pkg/tools/serialize"
)

type networkServiceEndpointRegistryServer struct {
	networkServiceEndpoints NetworkServiceEndpointSyncMap
	executor                serialize.Executor
	eventChannels           []chan *registry.NetworkServiceEndpoint
	eventChannelSize        int
}

func (n *networkServiceEndpointRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	n.networkServiceEndpoints.Store(r.Name, r)
	n.executor.AsyncExec(func() {
		for _, ch := range n.eventChannels {
			ch <- r
		}
	})
	return r, err
}

func (n *networkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	sendAllMatches := func(ns *registry.NetworkServiceEndpoint) error {
		var err error
		n.networkServiceEndpoints.Range(func(key string, value *registry.NetworkServiceEndpoint) bool {
			if matchutils.MatchNetworkServiceEndpoints(ns, value) {
				err = s.Send(value)
				return err == nil
			}
			return true
		})
		return err
	}
	if query.Watch {
		eventCh := make(chan *registry.NetworkServiceEndpoint, n.eventChannelSize)
		n.executor.AsyncExec(func() {
			n.eventChannels = append(n.eventChannels, eventCh)
		})
		err := sendAllMatches(query.NetworkServiceEndpoint)
		if err != nil {
			return err
		}
		for {
			select {
			case <-s.Context().Done():
				return s.Context().Err()
			case event := <-eventCh:
				if matchutils.MatchNetworkServiceEndpoints(query.NetworkServiceEndpoint, event) {
					if s.Context().Err() != nil {
						return err
					}
					if err := s.Send(event); err != nil {
						return err
					}
				}
			}
		}
	} else if err := sendAllMatches(query.NetworkServiceEndpoint); err != nil {
		return err
	}
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *networkServiceEndpointRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	n.networkServiceEndpoints.Delete(nse.Name)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func (n *networkServiceEndpointRegistryServer) setEventChannelSize(l int) {
	n.eventChannelSize = l
}

// NewNetworkServiceEndpointRegistryServer creates new memory based NetworkServiceEndpointRegistryServer
func NewNetworkServiceEndpointRegistryServer(options ...option) registry.NetworkServiceEndpointRegistryServer {
	r := &networkServiceEndpointRegistryServer{eventChannelSize: defaultEventChannelSize}
	for _, o := range options {
		o.apply(r)
	}
	return r
}
