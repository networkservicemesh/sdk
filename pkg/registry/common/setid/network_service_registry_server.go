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

package setid

import (
	"context"
	"strings"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/google/uuid"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

type setIDNetworkServiceEndpointRegistryServer struct{}

func (n *setIDNetworkServiceEndpointRegistryServer) Register(ctx context.Context, request *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	request.Name = nameOf(request)
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, request)
}

type setIDNetworkServiceEndpointRegistryFindServer struct {
	registry.NetworkServiceEndpointRegistry_FindServer
}

func (s *setIDNetworkServiceEndpointRegistryFindServer) Send(request *registry.NetworkServiceEndpoint) error {
	request = request.Clone()
	request.Name = nameOf(request)
	return s.NetworkServiceEndpointRegistry_FindServer.Send(request)
}

func (n *setIDNetworkServiceEndpointRegistryServer) Find(query *registry.NetworkServiceEndpointQuery, s registry.NetworkServiceEndpointRegistry_FindServer) error {
	return next.NetworkServiceEndpointRegistryServer(s.Context()).Find(query, &setIDNetworkServiceEndpointRegistryFindServer{NetworkServiceEndpointRegistry_FindServer: s})
}

func (n *setIDNetworkServiceEndpointRegistryServer) Unregister(ctx context.Context, request *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, request)
}

// NewNetworkServiceEndpointRegistryServer creates new instance of NetworkServiceRegistryServer which set the unique name for the endpoint on registration
func NewNetworkServiceEndpointRegistryServer() registry.NetworkServiceEndpointRegistryServer {
	return &setIDNetworkServiceEndpointRegistryServer{}
}

func nameOf(endpoint *registry.NetworkServiceEndpoint) string {
	if endpoint.Name != "" {
		return endpoint.Name
	}
	return strings.Join(append(endpoint.NetworkServiceNames, uuid.New().String()), "-")
}

var _ registry.NetworkServiceEndpointRegistryServer = &setIDNetworkServiceEndpointRegistryServer{}
