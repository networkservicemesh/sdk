// Copyright (c) 2022-2023 Cisco and/or its affiliates.
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

// Package interdomainbypass provides registry chain element that sets to outgoing NSE the public nsmgr-proxy and stores into the shared map the public nsmgr URL from the incoming endpoint.
package interdomainbypass

import (
	"context"
	"net/url"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
	"github.com/pkg/errors"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type interdomainBypassNSEServer struct {
	m *genericsync.Map[string, *url.URL]
	u *url.URL
}

type interdomainBypassNSEFindServer struct {
	m *genericsync.Map[string, *url.URL]
	u *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (n *interdomainBypassNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	originalURL := service.Url
	service.Url = n.u.String()

	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
	if err != nil {
		return nil, err
	}

	u, _ := url.Parse(originalURL)

	n.m.Store(service.GetName(), u)

	resp.Url = originalURL

	return resp, nil
}

func (n *interdomainBypassNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &interdomainBypassNSEFindServer{NetworkServiceEndpointRegistry_FindServer: server, m: n.m, u: n.u})
}

func (n *interdomainBypassNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	n.m.Delete(service.GetName())
	originalURL := service.Url
	service.Url = n.u.String()
	defer func() {
		service.Url = originalURL
	}()
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceEndpointRegistryServer creates new instance of interdomainbypass NSE server.
// It simply stores into passed stringurl.Map all incoming nse.Name:nse.URL entries.
// And sets passed URL for outgoing NSEs.
func NewNetworkServiceEndpointRegistryServer(m *genericsync.Map[string, *url.URL], u *url.URL) registry.NetworkServiceEndpointRegistryServer {
	if m == nil {
		panic("m can not be nil")
	}
	if u == nil {
		panic("u can not be nil")
	}
	return &interdomainBypassNSEServer{m: m, u: u}
}

func (s *interdomainBypassNSEFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	nseURL := nseResp.GetNetworkServiceEndpoint().GetUrl()
	u, err := url.Parse(nseURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse url %s", nseURL)
	}
	s.m.LoadOrStore(nseResp.GetNetworkServiceEndpoint().GetName(), u)
	nseResp.GetNetworkServiceEndpoint().Url = s.u.String()
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}
