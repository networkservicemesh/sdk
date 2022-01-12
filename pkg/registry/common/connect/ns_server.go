// Copyright (c) 2020-2022 Doc.ai and/or its affiliates.
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

package connect

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

type connectNSServer struct {
	ctx           context.Context
	clientOptions []Option

	nsInfos  nsInfoMap
	clients  nsClientMap
	executor multiexecutor.MultiExecutor
}

type nsInfo struct {
	clientURL *url.URL
	client    *nsClient
}

type nsClient struct {
	client  registry.NetworkServiceRegistryClient
	count   int
	onClose context.CancelFunc
}

// NewNetworkServiceRegistryServer - server chain element that creates client subchains and requests them selecting by
//             clienturlctx.ClientURL(ctx)
func NewNetworkServiceRegistryServer(
	ctx context.Context,
	clientOptions ...Option,
) registry.NetworkServiceRegistryServer {
	return &connectNSServer{
		ctx:           ctx,
		clientOptions: clientOptions,
	}
}

func (s *connectNSServer) Register(ctx context.Context, ns *registry.NetworkService) (*registry.NetworkService, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming service: %+v", ns)
	}

	_, loaded := s.nsInfos.Load(ns.Name)

	c := s.client(ctx, ns)
	reg, err := c.client.Register(ctx, ns)
	if err != nil {
		if !loaded {
			s.closeClient(c, clientURL.String())
		}
		return nil, err
	}

	s.nsInfos.Store(ns.Name, &nsInfo{
		clientURL: clientURL,
		client:    c,
	})

	return reg, nil
}

func (s *connectNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	clientURL := clienturlctx.ClientURL(server.Context())
	if clientURL == nil {
		return errors.Errorf("clientURL not found for incoming query: %+v", query)
	}

	c := s.client(server.Context(), nil)

	err := adapters.NetworkServiceClientToServer(c.client).Find(query, server)

	s.closeClient(c, clientURL.String())

	return err
}

func (s *connectNSServer) Unregister(ctx context.Context, ns *registry.NetworkService) (*empty.Empty, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming service: %+v", ns)
	}

	c := s.client(ctx, ns)

	_, err := c.client.Unregister(ctx, ns)

	s.closeClient(c, clientURL.String())
	s.nsInfos.Delete(ns.Name)

	return new(empty.Empty), err
}

func (s *connectNSServer) client(ctx context.Context, ns *registry.NetworkService) *nsClient {
	clientURL := clienturlctx.ClientURL(ctx)

	if ns != nil {
		// First check if we have already registered on some clientURL with this ns.Name.
		if info, ok := s.nsInfos.Load(ns.Name); ok {
			if *info.clientURL == *clientURL {
				return info.client
			}

			// For some reason we have changed the clientURL, so we need to close the existing client.
			s.closeClient(info.client, info.clientURL.String())
		}
	}

	var c *nsClient
	<-s.executor.AsyncExec(clientURL.String(), func() {
		// Fast path if we already have client for the clientURL and we should not reconnect, use it.
		var loaded bool
		c, loaded = s.clients.Load(clientURL.String())
		if !loaded {
			// If not, create and LoadOrStore a new one.
			c = s.newClient(clientURL)
			s.clients.Store(clientURL.String(), c)
		}
		c.count++
	})
	return c
}

func (s *connectNSServer) newClient(clientURL *url.URL) *nsClient {
	ctx, cancel := context.WithCancel(s.ctx)
	return &nsClient{
		client:  NewNetworkServiceRegistryClient(ctx, clientURL, s.clientOptions...),
		count:   0,
		onClose: cancel,
	}
}

func (s *connectNSServer) closeClient(c *nsClient, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		c.count--
		if c.count == 0 {
			if loadedClient, ok := s.clients.Load(clientURL); ok && c == loadedClient {
				s.clients.Delete(clientURL)
			}
			c.onClose()
		}
	})
}

func (s *connectNSServer) deleteClient(c *nsClient, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		if loadedClient, ok := s.clients.Load(clientURL); ok && c == loadedClient {
			s.clients.Delete(clientURL)
		}
		c.onClose()
	})
}
