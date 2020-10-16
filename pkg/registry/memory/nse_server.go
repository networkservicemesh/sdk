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
	"errors"
	"io"

	"github.com/google/uuid"

	"github.com/golang/protobuf/ptypes/timestamp"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

type networkServiceEndpointRegistryServer struct {
	networkServiceEndpoints NetworkServiceEndpointSyncMap
	executor                serialize.Executor
	eventChannels           map[string]chan *registry.NetworkServiceEndpoint
	eventChannelSize        int
}

func (n *networkServiceEndpointRegistryServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}
	n.networkServiceEndpoints.Store(r.Name, r.Clone())
	n.sendEvent(r)
	return r, err
}

func (n *networkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	sendAllMatches := func(ns *registry.NetworkServiceEndpoint) error {
		var err error
		n.networkServiceEndpoints.Range(func(key string, value *registry.NetworkServiceEndpoint) bool {
			if matchutils.MatchNetworkServiceEndpoints(ns, value) {
				err = s.Send(value.Clone())
				return err == nil
			}
			return true
		})
		return err
	}
	if query.Watch {
		eventCh := make(chan *registry.NetworkServiceEndpoint, n.eventChannelSize)
		id := uuid.New().String()
		n.executor.AsyncExec(func() {
			n.eventChannels[id] = eventCh
		})
		defer n.executor.AsyncExec(func() {
			delete(n.eventChannels, id)
		})
		err := sendAllMatches(query.NetworkServiceEndpoint)
		if err != nil {
			return err
		}
		notifyChannel := func() error {
			select {
			case <-s.Context().Done():
				return io.EOF
			case event := <-eventCh:
				if matchutils.MatchNetworkServiceEndpoints(query.NetworkServiceEndpoint, event) {
					if s.Context().Err() != nil {
						return io.EOF
					}
					if err := s.Send(event); err != nil {
						return err
					}
				}
				return nil
			}
		}
		for {
			err := notifyChannel()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return err
			}
		}
	} else if err := sendAllMatches(query.NetworkServiceEndpoint); err != nil {
		return err
	}
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, s)
}

func (n *networkServiceEndpointRegistryServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	n.networkServiceEndpoints.Delete(nse.Name)
	if nse.ExpirationTime == nil {
		nse.ExpirationTime = &timestamp.Timestamp{}
	}
	<-n.executor.AsyncExec(func() {
		nse.ExpirationTime.Seconds = -1
	})
	n.sendEvent(nse)
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}

func (n *networkServiceEndpointRegistryServer) setEventChannelSize(l int) {
	n.eventChannelSize = l
}

func (n *networkServiceEndpointRegistryServer) sendEvent(nse *registry.NetworkServiceEndpoint) {
	n.executor.AsyncExec(func() {
		for _, ch := range n.eventChannels {
			ch <- nse
		}
	})
}

// NewNetworkServiceEndpointRegistryServer creates new memory based NetworkServiceEndpointRegistryServer
func NewNetworkServiceEndpointRegistryServer(options ...Option) registry.NetworkServiceEndpointRegistryServer {
	r := &networkServiceEndpointRegistryServer{
		eventChannelSize: defaultEventChannelSize,
		eventChannels:    make(map[string]chan *registry.NetworkServiceEndpoint),
	}
	for _, o := range options {
		o.apply(r)
	}
	return r
}
