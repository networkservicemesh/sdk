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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/edwarnicke/serialize"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/matchutils"
)

type networkServiceRegistryServer struct {
	networkServices  NetworkServiceSyncMap
	executor         serialize.Executor
	eventChannels    map[string]chan *registry.NetworkService
	eventChannelSize int
}

func (n *networkServiceRegistryServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	r, err := next.NetworkServiceRegistryServer(ctx).Register(ctx, ns)
	if err != nil {
		return nil, err
	}
	n.networkServices.Store(r.Name, r.Clone())
	n.executor.AsyncExec(func() {
		for _, ch := range n.eventChannels {
			ch <- r
		}
	})
	return r, nil
}

func (n *networkServiceRegistryServer) Find(query *registry.NetworkServiceQuery, s registry.NetworkServiceRegistry_FindServer) error {
	sendAllMatches := func(ns *registry.NetworkService) error {
		var err error
		n.networkServices.Range(func(key string, value *registry.NetworkService) bool {
			if matchutils.MatchNetworkServices(ns, value) {
				err = s.Send(value.Clone())
				return err == nil
			}
			return true
		})
		return err
	}
	if query.Watch {
		eventCh := make(chan *registry.NetworkService, n.eventChannelSize)
		id := uuid.New().String()
		n.executor.AsyncExec(func() {
			n.eventChannels[id] = eventCh
		})
		defer n.executor.AsyncExec(func() {
			delete(n.eventChannels, id)
		})
		err := sendAllMatches(query.NetworkService)
		if err != nil {
			return err
		}
		notifyChannel := func() error {
			select {
			case <-s.Context().Done():
				return io.EOF
			case event := <-eventCh:
				if matchutils.MatchNetworkServices(query.NetworkService, event) {
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
	} else if err := sendAllMatches(query.NetworkService); err != nil {
		return err
	}
	return next.NetworkServiceRegistryServer(s.Context()).Find(query, s)
}

func (n *networkServiceRegistryServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	n.networkServices.Delete(ns.Name)
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, ns)
}

func (n *networkServiceRegistryServer) setEventChannelSize(l int) {
	n.eventChannelSize = l
}

// NewNetworkServiceRegistryServer creates new memory based NetworkServiceRegistryServer
func NewNetworkServiceRegistryServer(options ...Option) registry.NetworkServiceRegistryServer {
	r := &networkServiceRegistryServer{
		eventChannelSize: defaultEventChannelSize,
		eventChannels:    make(map[string]chan *registry.NetworkService),
	}
	for _, o := range options {
		o.apply(r)
	}
	return r
}
