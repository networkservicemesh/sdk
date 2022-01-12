// Copyright (c) 2022 Doc.ai and/or its affiliates.
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

// Package storeurl provides chain element that stores incoming NSE URLs into map
package storeurl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
	"github.com/networkservicemesh/sdk/pkg/tools/stringurl"
)

type storeurl struct {
	m *stringurl.Map
}

type urlstockFindServer struct {
	m *stringurl.Map
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (n *storeurl) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (n *storeurl) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &urlstockFindServer{NetworkServiceEndpointRegistry_FindServer: server, m: n.m})
}

func (n *storeurl) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceEndpointRegistryServer creates new instance of storeurl NSE server
func NewNetworkServiceEndpointRegistryServer(m *stringurl.Map) registry.NetworkServiceEndpointRegistryServer {
	if m == nil {
		panic("m can not be nil")
	}
	return &storeurl{m: m}
}

func (s *urlstockFindServer) Send(nseResp *registry.NetworkServiceEndpointResponse) error {
	u, err := url.Parse(nseResp.NetworkServiceEndpoint.Url)
	if err != nil {
		return err
	}
	s.m.Store(nseResp.NetworkServiceEndpoint.Name, u)
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nseResp)
}
