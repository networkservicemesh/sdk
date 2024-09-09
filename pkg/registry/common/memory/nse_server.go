// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023 Cisco and/or its affiliates.
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
	"io"

	"github.com/edwarnicke/genericsync"
	"github.com/edwarnicke/serialize"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

type memoryNSEServer struct {
	networkServiceEndpoints genericsync.Map[string, *registry.NetworkServiceEndpoint]
	executor                serialize.Executor
	eventChannels           map[string]chan *registry.NetworkServiceEndpointResponse
	eventChannelSize        int
}

// NewNetworkServiceEndpointRegistryServer creates new memory based NetworkServiceEndpointRegistryServer.
func NewNetworkServiceEndpointRegistryServer(options ...Option) registry.NetworkServiceEndpointRegistryServer {
	s := &memoryNSEServer{
		eventChannelSize: defaultEventChannelSize,
		eventChannels:    make(map[string]chan *registry.NetworkServiceEndpointResponse),
	}
	for _, o := range options {
		o.apply(s)
	}
	return s
}

func (s *memoryNSEServer) setEventChannelSize(l int) {
	s.eventChannelSize = l
}

func (s *memoryNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	r, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
	if err != nil {
		return nil, err
	}

	s.networkServiceEndpoints.Store(r.GetName(), r.Clone())

	s.sendEvent(&registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: r})

	return r, nil
}

func (s *memoryNSEServer) sendEvent(event *registry.NetworkServiceEndpointResponse) {
	event = event.Clone()
	s.executor.AsyncExec(func() {
		for _, ch := range s.eventChannels {
			ch <- event.Clone()
		}
	})
}

func (s *memoryNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	if !query.GetWatch() {
		for _, nse := range s.allMatches(query) {
			nseResp := &registry.NetworkServiceEndpointResponse{
				NetworkServiceEndpoint: nse,
			}
			if err := server.Send(nseResp); err != nil {
				return errors.Wrapf(err, "NetworkServiceRegistry find server failed to send a response %s", nseResp.String())
			}
		}
		return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
	}

	if err := next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server); err != nil {
		return err
	}

	eventCh := make(chan *registry.NetworkServiceEndpointResponse, s.eventChannelSize)
	id := uuid.New().String()

	s.executor.AsyncExec(func() {
		s.eventChannels[id] = eventCh
		for _, entity := range s.allMatches(query) {
			eventCh <- &registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: entity}
		}
	})
	defer s.closeEventChannel(id, eventCh)

	var err error
	for ; err == nil; err = s.receiveEvent(query, server, eventCh) {
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func (s *memoryNSEServer) allMatches(query *registry.NetworkServiceEndpointQuery) (matches []*registry.NetworkServiceEndpoint) {
	s.networkServiceEndpoints.Range(func(_ string, nse *registry.NetworkServiceEndpoint) bool {
		if matchutils.MatchNetworkServiceEndpoints(query.GetNetworkServiceEndpoint(), nse) {
			matches = append(matches, nse.Clone())
		}
		return true
	})
	return matches
}

func (s *memoryNSEServer) closeEventChannel(id string, eventCh <-chan *registry.NetworkServiceEndpointResponse) {
	ctx, cancel := context.WithCancel(context.Background())

	s.executor.AsyncExec(func() {
		delete(s.eventChannels, id)
		cancel()
	})

	for {
		select {
		case <-ctx.Done():
			return
		case <-eventCh:
		}
	}
}

func (s *memoryNSEServer) receiveEvent(
	query *registry.NetworkServiceEndpointQuery,
	server registry.NetworkServiceEndpointRegistry_FindServer,
	eventCh <-chan *registry.NetworkServiceEndpointResponse,
) error {
	select {
	case <-server.Context().Done():
		return errors.WithStack(io.EOF)
	case event := <-eventCh:
		if matchutils.MatchNetworkServiceEndpoints(query.GetNetworkServiceEndpoint(), event.GetNetworkServiceEndpoint()) {
			if err := server.Send(event); err != nil {
				if server.Context().Err() != nil {
					return errors.WithStack(io.EOF)
				}
				return errors.Wrapf(err, "NetworkServiceRegistry find server failed to send a response %s", event.String())
			}
		}
		return nil
	}
}

func (s *memoryNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if unregisterNSE, ok := s.networkServiceEndpoints.LoadAndDelete(nse.GetName()); ok {
		unregisterNSE = unregisterNSE.Clone()
		s.sendEvent(&registry.NetworkServiceEndpointResponse{NetworkServiceEndpoint: unregisterNSE, Deleted: true})
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
