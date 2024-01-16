// Copyright (c) 2024 Cisco and/or its affiliates.
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

// Package replaceurl provides a chain element that replaces the URL of found NSEs.
package replaceurl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type findReplaceNSEURLServer struct {
	value *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *findReplaceNSEURLServer) Send(resp *registry.NetworkServiceEndpointResponse) error {
	if s.value != nil {
		resp.GetNetworkServiceEndpoint().Url = s.value.String()
	}
	return s.NetworkServiceEndpointRegistry_FindServer.Send(resp)
}

type replaceNSEURLServer struct {
	value *url.URL
}

// NewNetworkServiceEndpointRegistryServer ccreates a new NetworkServiceEndpointRegistryServer that replaces urls of found NSEs
func NewNetworkServiceEndpointRegistryServer(v *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return &replaceNSEURLServer{value: v}
}

func (n *replaceNSEURLServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (n *replaceNSEURLServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &findReplaceNSEURLServer{
		NetworkServiceEndpointRegistry_FindServer: server,
		value: n.value,
	})
}

func (n *replaceNSEURLServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
