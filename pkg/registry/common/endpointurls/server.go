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

// Package endpointurls provides registry.NetworkServiceEndpointRegistryServer that can be injected in the chain of registry.NetworkServiceEndpointRegistryServer to get an actual nses of registry.NetworkServiceEndpoint URLs.
package endpointurls

import (
	"context"
	"net/url"

	"github.com/edwarnicke/genericsync"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type endpointURLsServer struct {
	nses *genericsync.Map[url.URL, string]
}

func (e *endpointURLsServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	if u, err := url.Parse(endpoint.GetUrl()); err == nil {
		e.nses.Store(*u, endpoint.GetName())
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (e *endpointURLsServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (e *endpointURLsServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	if u, err := url.Parse(endpoint.GetUrl()); err == nil {
		e.nses.Delete(*u)
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}

// NewNetworkServiceEndpointRegistryServer returns new registry.NetworkServiceEndpointRegistryServer with injected endpoint urls nses.
func NewNetworkServiceEndpointRegistryServer(m *genericsync.Map[url.URL, string]) registry.NetworkServiceEndpointRegistryServer {
	return &endpointURLsServer{nses: m}
}
