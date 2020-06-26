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

// Package seturl implements a chain elemenet to set url to registered endpoints.
package seturl

import (
	"context"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"
)

type setMgrServer struct {
	url string
}

type setNSMgrURLFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
	url string
}

func (s *setNSMgrURLFindServer) Send(endpoint *registry.NetworkServiceEndpoint) error {
	endpoint.Url = s.url
	return s.NetworkServiceEndpointRegistry_FindServer.Send(endpoint)
}

func (s *setMgrServer) Register(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	endpoint.Url = s.url
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, endpoint)
}

func (s *setMgrServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, &setNSMgrURLFindServer{url: s.url, NetworkServiceEndpointRegistry_FindServer: server})
}

func (s *setMgrServer) Unregister(ctx context.Context, endpoint *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	endpoint.Url = s.url
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, endpoint)
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceEndpointRegistryServer which set the passed NSMgr url
func NewNetworkServiceEndpointRegistryServer(u string) registry.NetworkServiceEndpointRegistryServer {
	return &setMgrServer{
		url: u,
	}
}
