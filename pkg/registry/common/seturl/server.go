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

// Package seturl provides chain elements for changing URLs on registration / unregistration
package seturl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setNSEURLServer struct {
	value *url.URL
}

// NewNetworkServiceEndpointRegistryServer creates a new NetworkServiceEndpointRegistryServer that replaces URL on register and unregister
func NewNetworkServiceEndpointRegistryServer(v *url.URL) registry.NetworkServiceEndpointRegistryServer {
	return &setNSEURLServer{value: v}
}

func (n *setNSEURLServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	nse.Url = n.value.String()
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)
}

func (n *setNSEURLServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (n *setNSEURLServer) Unregister(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	nse.Url = n.value.String()
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, nse)
}
