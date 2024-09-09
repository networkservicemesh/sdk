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

// Package switchcase provides chain elements acting like a switch-case statement, selecting a chain element with first
// succeed condition
package switchcase

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/networkservicemesh/api/pkg/api/registry"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NSServerCase repsenets NetworkService case for servers.
type NSServerCase struct {
	Condition func(context.Context, *registry.NetworkService) bool
	Action    registry.NetworkServiceRegistryServer
}

type switchCaseNSServer struct {
	cases []NSServerCase
}

func (n *switchCaseNSServer) Register(ctx context.Context, service *registry.NetworkService) (*registry.NetworkService, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, service) {
			return c.Action.Register(ctx, service)
		}
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, service)
}

func (n *switchCaseNSServer) Find(query *registry.NetworkServiceQuery, server registry.NetworkServiceRegistry_FindServer) error {
	for _, c := range n.cases {
		if c.Condition(server.Context(), query.GetNetworkService()) {
			return c.Action.Find(query, server)
		}
	}
	return next.NetworkServiceRegistryServer(server.Context()).Find(query, server)
}

func (n *switchCaseNSServer) Unregister(ctx context.Context, service *registry.NetworkService) (*empty.Empty, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, service) {
			return c.Action.Unregister(ctx, service)
		}
	}
	return next.NetworkServiceRegistryServer(ctx).Unregister(ctx, service)
}

// NewNetworkServiceRegistryServer - returns a new switchcase server.
func NewNetworkServiceRegistryServer(cases ...NSServerCase) registry.NetworkServiceRegistryServer {
	for index, c := range cases {
		if c.Action == nil {
			panic(fmt.Sprintf("index: %v, %v.Action is nil", index, c))
		}
		if c.Condition == nil {
			panic(fmt.Sprintf("index: %v, %v.Condition is nil", index, c))
		}
	}

	return &switchCaseNSServer{
		cases: cases,
	}
}
