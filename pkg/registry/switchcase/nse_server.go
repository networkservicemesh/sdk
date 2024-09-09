// Copyright (c) 2022  Doc.ai and/or its affiliates.
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

package switchcase

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NSEServerCase repsenets NetworkServiceEndpoint case for servers.
type NSEServerCase struct {
	Condition func(context.Context, *registry.NetworkServiceEndpoint) bool
	Action    registry.NetworkServiceEndpointRegistryServer
}

type switchCaseNSEServer struct {
	cases []NSEServerCase
}

func (n *switchCaseNSEServer) Register(ctx context.Context, service *registry.NetworkServiceEndpoint) (*registry.NetworkServiceEndpoint, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, service) {
			return c.Action.Register(ctx, service)
		}
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Register(ctx, service)
}

func (n *switchCaseNSEServer) Find(query *registry.NetworkServiceEndpointQuery, server registry.NetworkServiceEndpointRegistry_FindServer) error {
	for _, c := range n.cases {
		if c.Condition(server.Context(), query.GetNetworkServiceEndpoint()) {
			return c.Action.Find(query, server)
		}
	}
	return next.NetworkServiceEndpointRegistryServer(server.Context()).Find(query, server)
}

func (n *switchCaseNSEServer) Unregister(ctx context.Context, service *registry.NetworkServiceEndpoint) (*empty.Empty, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, service) {
			return c.Action.Unregister(ctx, service)
		}
	}
	return next.NetworkServiceEndpointRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceEndpointRegistryServer - returns a new switchcase server.
func NewNetworkServiceEndpointRegistryServer(cases ...NSEServerCase) registry.NetworkServiceEndpointRegistryServer {
	for index, c := range cases {
		if c.Action == nil {
			panic(fmt.Sprintf("index: %v, %v.Action is nil", index, c))
		}
		if c.Condition == nil {
			panic(fmt.Sprintf("index: %v, %v.Condition is nil", index, c))
		}
	}

	return &switchCaseNSEServer{
		cases: cases,
	}
}
