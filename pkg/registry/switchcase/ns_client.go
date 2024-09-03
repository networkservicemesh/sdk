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
	"google.golang.org/grpc"

	"github.com/networkservicemesh/sdk/pkg/registry/core/next"
)

// NSClientCase repsenets NetworkService case for clients.
type NSClientCase struct {
	Condition func(context.Context, *registry.NetworkService) bool
	Action    registry.NetworkServiceRegistryClient
}

type switchCaseNSClient struct {
	cases []NSClientCase
}

func (n *switchCaseNSClient) Register(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*registry.NetworkService, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, in) {
			return c.Action.Register(ctx, in)
		}
	}
	return next.NetworkServiceRegistryServer(ctx).Register(ctx, in)
}

func (n *switchCaseNSClient) Find(ctx context.Context, in *registry.NetworkServiceQuery, opts ...grpc.CallOption) (registry.NetworkServiceRegistry_FindClient, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, in.GetNetworkService()) {
			return c.Action.Find(ctx, in)
		}
	}
	return next.NetworkServiceRegistryClient(ctx).Find(ctx, in, opts...)
}

func (n *switchCaseNSClient) Unregister(ctx context.Context, in *registry.NetworkService, opts ...grpc.CallOption) (*empty.Empty, error) {
	for _, c := range n.cases {
		if c.Condition(ctx, in) {
			return c.Action.Unregister(ctx, in)
		}
	}
	return next.NetworkServiceRegistryClient(ctx).Unregister(ctx, in, opts...)
}

// NewNetworkServiceRegistryClient - returns a new switchcase client.
func NewNetworkServiceRegistryClient(cases ...NSClientCase) registry.NetworkServiceRegistryClient {
	for index, c := range cases {
		if c.Action == nil {
			panic(fmt.Sprintf("index: %v, %v.Action is nil", index, c))
		}
		if c.Condition == nil {
			panic(fmt.Sprintf("index: %v, %v.Condition is nil", index, c))
		}
	}

	return &switchCaseNSClient{
		cases: cases,
	}
}
