// Copyright (c) 2021 Doc.ai and/or its affiliates.
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

// Package seturl provides registry.NetworkServiceEndpointRegistryServer that sets passed url for each found nse
package seturl

import (
	"context"
	"net/url"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type seturlNSEServer struct {
	u *url.URL
}

func (s *setURLNSEServer) Send(nse *registry.NetworkServiceEndpointResponse) error {
	nse.NetworkServiceEndpoint.Url = s.u.String()
	return s.NetworkServiceEndpointRegistry_FindServer.Send(nse)
}

type setURLNSEServer struct {
	u *url.URL
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (n *seturlNSEServer) Register(ctx context.Context, nse *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	u := nse.Url
	nse.Url = n.u.String()
	resp, err := next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, nse)

	if resp != nil {
		resp.Url = u
	}

	return resp, err
}

func (n *seturlNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &setURLNSEServer{NetworkServiceEndpointRegistry_FindServer: server, u: n.u})
}

func (n *seturlNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	service.Url = n.u.String()
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceEndpointRegistryServer creates a new seturl registry.NetworkServiceEndpointRegistryServer
func NewNetworkServiceEndpointRegistryServer(u *url.URL) registry.NetworkServiceEndpointRegistryServer {
	if u == nil {
		panic("u can not be nil")
	}
	return &seturlNSEServer{u: u}
}
