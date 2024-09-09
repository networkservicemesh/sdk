// Copyright (c) 2020-2021 Doc.ai and/or its affiliates.
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

type memoryNSServer struct {
	networkServices  genericsync.Map[string, *registry.NetworkService]
	executor         serialize.Executor
	eventChannels    map[string]chan *registry.NetworkService
	eventChannelSize int
}

// NewNetworkServiceRegistryServer creates new memory based NetworkServiceRegistryServer.
func NewNetworkServiceRegistryServer(options ...Option) registry.NetworkServiceRegistryServer {
	s := &memoryNSServer{
		eventChannelSize: defaultEventChannelSize,
		eventChannels:    make(map[string]chan *registry.NetworkService),
	}
	for _, o := range options {
		o.apply(s)
	}
	return s
}

func (s *memoryNSServer) setEventChannelSize(l int) {
	s.eventChannelSize = l
}

func (s *memoryNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	r, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}

	s.networkServices.Store(r.GetName(), r.Clone())

	s.sendEvent(r)

	return r, nil
}

func (s *memoryNSServer) sendEvent(event *registry.NetworkService) {
	event = event.Clone()
	s.executor.AsyncExec(func() {
		for _, ch := range s.eventChannels {
			ch <- event.Clone()
		}
	})
}

func (s *memoryNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	if !query.GetWatch() {
		for _, ns := range s.allMatches(query) {
			nsResp := &registry.NetworkServiceResponse{
				NetworkService: ns,
			}

			if err := server.Send(nsResp); err != nil {
				return errors.Wrapf(err, "NetworkServiceRegistry find server failed to send a response %s", nsResp.String())
			}
		}
		return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
	}

	eventCh := make(chan *registry.NetworkService, s.eventChannelSize)
	id := uuid.New().String()

	s.executor.AsyncExec(func() {
		s.eventChannels[id] = eventCh
		for _, entity := range s.allMatches(query) {
			eventCh <- entity
		}
	})
	defer s.closeEventChannel(id, eventCh)

	var err error
	for ; err == nil; err = s.receiveEvent(query, server, eventCh) {
	}
	if !errors.Is(err, io.EOF) {
		return err
	}
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (s *memoryNSServer) allMatches(query *registry.NetworkServiceQuery) (matches []*registry.NetworkService) {
	s.networkServices.Range(func(_ string, ns *registry.NetworkService) bool {
		if matchutils.MatchNetworkServices(query.GetNetworkService(), ns) {
			matches = append(matches, ns.Clone())
		}
		return true
	})
	return matches
}

func (s *memoryNSServer) closeEventChannel(id string, eventCh <-chan *registry.NetworkService) {
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

func (s *memoryNSServer) receiveEvent(
	query *registry.NetworkServiceQuery,
	server registry.NetworkServiceRegistry_FindServer,
	eventCh <-chan *registry.NetworkService,
) error {
	select {
	case <-server.Context().Done():
		return errors.WithStack(io.EOF)
	case event := <-eventCh:
		if matchutils.MatchNetworkServices(query.GetNetworkService(), event) {
			nse := &registry.NetworkServiceResponse{
				NetworkService: event,
			}

			if err := server.Send(nse); err != nil {
				if server.Context().Err() != nil {
					return errors.WithStack(io.EOF)
				}
				return errors.Wrapf(err, "NetworkServiceRegistry find server failed to send a response %s", nse.String())
			}
		}
		return nil
	}
}

func (s *memoryNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	s.networkServices.Delete(ns.GetName())

	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}
