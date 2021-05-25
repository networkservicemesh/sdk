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

package connect

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/adapters"
	"github.com/networkservicemesh/sdk/pkg/tools/clienturlctx"
	"github.com/networkservicemesh/sdk/pkg/tools/multiexecutor"
)

// NSEClientFactory is a NSE client chain supplier func type
type NSEClientFactory func(ctx context.Context, cc grpc.ClientConnInterface) registry.NetworkServiceEndpointRegistryClient

type connectNSEServer struct {
	ctx               context.Context
	clientFactory     NSEClientFactory
	clientDialOptions []grpc.DialOption

	nseInfos nseInfoMap
	clients  nseClientMap
	executor multiexecutor.MultiExecutor
}

type nseInfo struct {
	clientURL *url.URL
	client    *nseClient
}

type nseClient struct {
	client  *connectNSEClient
	count   int
	onClose context.CancelFunc
}

// NewNetworkServiceEndpointRegistryServer - server chain element that creates client subchains and requests them selecting by
//             clienturlctx.ClientURL(ctx)
func NewNetworkServiceEndpointRegistryServer(
	ctx context.Context,
	clientFactory NSEClientFactory,
	clientDialOptions ...grpc.DialOption,
) registry.NetworkServiceEndpointRegistryServer {
	return &connectNSEServer{
		ctx:               ctx,
		clientFactory:     clientFactory,
		clientDialOptions: clientDialOptions,
	}
}

func (s *connectNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming endpoint: %+v", nse)
	}

	_, loaded := s.nseInfos.Load(nse.Name)

	c := s.client(ctx, nse)
	reg, err := c.client.Register(ctx, nse)
	if err != nil {
		if !loaded {
			s.closeClient(c, clientURL.String())
		}

		// Close current client chain if gRPC connection was closed
		if c.client.ctx.Err() != nil {
			s.deleteClient(c, clientURL.String())
			s.nseInfos.Delete(nse.Name)
		}

		return nil, err
	}

	s.nseInfos.Store(nse.Name, &nseInfo{
		clientURL: clientURL,
		client:    c,
	})

	return reg, nil
}

func (s *connectNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	clientURL := clienturlctx.ClientURL(server.Context())
	if clientURL == nil {
		return errors.Errorf("clientURL not found for incoming query: %+v", query)
	}

	c := s.client(server.Context(), nil)

	return adapters.NetworkServiceEndpointClientToServer(c.client).Find(query, &connectNSEFindServer{
		client:           c,
		clientURL:        clientURL.String(),
		connectNSEServer: s,
		NetworkServiceEndpointRegistry_FindServer: server,
	})
}

func (s *connectNSEServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	clientURL := clienturlctx.ClientURL(ctx)
	if clientURL == nil {
		return nil, errors.Errorf("clientURL not found for incoming endpoint: %+v", nse)
	}

	c := s.client(ctx, nse)
	_, err := c.client.Unregister(ctx, nse)
	if err != nil && c.client.ctx.Err() != nil {
		s.deleteClient(c, clientURL.String())
	} else {
		s.closeClient(c, clientURL.String())
	}

	s.nseInfos.Delete(nse.Name)

	return new(empty.Empty), err
}

func (s *connectNSEServer) client(ctx context.Context, nse *registry.NetworkServiceEndpoint) *nseClient {
	clientURL := clienturlctx.ClientURL(ctx)

	if nse != nil {
		// First check if we have already registered on some clientURL with this nse.Name.
		if info, ok := s.nseInfos.Load(nse.Name); ok {
			if *info.clientURL == *clientURL {
				return info.client
			}

			// For some reason we have changed the clientURL, so we need to close the existing client.
			s.closeClient(info.client, info.clientURL.String())
		}
	}

	var c *nseClient
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

func (s *connectNSEServer) newClient(clientURL *url.URL) *nseClient {
	ctx, cancel := context.WithCancel(s.ctx)
	return &nseClient{
		client: &connectNSEClient{
			ctx:           clienturlctx.WithClientURL(ctx, clientURL),
			clientFactory: s.clientFactory,
			dialOptions:   s.clientDialOptions,
		},
		count:   0,
		onClose: cancel,
	}
}

func (s *connectNSEServer) closeClient(c *nseClient, clientURL string) {
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

func (s *connectNSEServer) deleteClient(c *nseClient, clientURL string) {
	<-s.executor.AsyncExec(clientURL, func() {
		if loadedClient, ok := s.clients.Load(clientURL); ok && c == loadedClient {
			s.clients.Delete(clientURL)
		}
		c.onClose()
	})
}
